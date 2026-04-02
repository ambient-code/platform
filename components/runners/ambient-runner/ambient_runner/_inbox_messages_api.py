from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Iterator, Optional

import grpc

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class InboxMessage:
    id: str
    agent_id: str
    from_agent_id: Optional[str]
    from_name: Optional[str]
    body: str
    read: Optional[bool]
    created_at: Optional[datetime]
    updated_at: Optional[datetime]

    @classmethod
    def _from_proto(cls, pb: object) -> InboxMessage:
        def _ts(ts: object) -> Optional[datetime]:
            if ts is None:
                return None
            try:
                return datetime.fromtimestamp(
                    ts.seconds + ts.nanos / 1e9, tz=timezone.utc
                )
            except Exception:
                return None

        return cls(
            id=getattr(pb, "id", ""),
            agent_id=getattr(pb, "agent_id", ""),
            from_agent_id=getattr(pb, "from_agent_id", None) or None,
            from_name=getattr(pb, "from_name", None) or None,
            body=getattr(pb, "body", ""),
            read=getattr(pb, "read", None),
            created_at=_ts(getattr(pb, "created_at", None)),
            updated_at=_ts(getattr(pb, "updated_at", None)),
        )


class InboxMessagesAPI:
    """gRPC client wrapper for InboxService.WatchInboxMessages (server-streaming, watch-only)."""

    _WATCH_METHOD = "/ambient.v1.InboxService/WatchInboxMessages"

    def __init__(self, channel: grpc.Channel, token: str = "") -> None:
        self._metadata = [("authorization", f"Bearer {token}")] if token else []
        self._watch_rpc = channel.unary_stream(
            self._WATCH_METHOD,
            request_serializer=_WatchInboxRequest.SerializeToString,
            response_deserializer=_InboxMessageProto.FromString,
        )

    def watch(
        self,
        agent_id: str,
        *,
        timeout: Optional[float] = None,
    ) -> Iterator[InboxMessage]:
        """Stream live inbox messages for an agent.

        The server delivers only messages created AFTER the subscription
        begins — there is no replay cursor (unlike WatchSessionMessages).
        """
        logger.info(
            "[GRPC INBOX WATCH←] Starting WatchInboxMessages: agent_id=%s",
            agent_id,
        )
        req = _WatchInboxRequest()
        req.agent_id = agent_id
        stream = self._watch_rpc(req, timeout=timeout, metadata=self._metadata)
        msg_count = 0
        for pb in stream:
            msg = InboxMessage._from_proto(pb)
            msg_count += 1
            logger.info(
                "[GRPC INBOX WATCH←] Message #%d received: agent_id=%s inbox_id=%s from=%s body_len=%d",
                msg_count,
                agent_id,
                msg.id,
                msg.from_name or msg.from_agent_id or "system",
                len(msg.body),
            )
            yield msg
        logger.info(
            "[GRPC INBOX WATCH←] Stream ended: agent_id=%s total_messages=%d",
            agent_id,
            msg_count,
        )


# ---------------------------------------------------------------------------
# Minimal inline proto message classes (no generated _pb2 dependency).
# Mirrors the hand-rolled encoding in _session_messages_api.py.
# ---------------------------------------------------------------------------


def _encode_string(field_number: int, value: str) -> bytes:
    encoded = value.encode("utf-8")
    tag = (field_number << 3) | 2
    return _varint(tag) + _varint(len(encoded)) + encoded


def _varint(value: int) -> bytes:
    bits = value & 0x7F
    value >>= 7
    result = b""
    while value:
        result += bytes([0x80 | bits])
        bits = value & 0x7F
        value >>= 7
    result += bytes([bits])
    return result


def _decode_varint(data: bytes, pos: int) -> tuple[int, int]:
    result = 0
    shift = 0
    while True:
        b = data[pos]
        pos += 1
        result |= (b & 0x7F) << shift
        if not (b & 0x80):
            return result, pos
        shift += 7


def _decode_string(data: bytes, pos: int) -> tuple[str, int]:
    length, pos = _decode_varint(data, pos)
    return data[pos : pos + length].decode("utf-8", errors="replace"), pos + length


class _WatchInboxRequest:
    def __init__(self) -> None:
        self.agent_id: str = ""

    def SerializeToString(self) -> bytes:
        out = b""
        if self.agent_id:
            out += _encode_string(1, self.agent_id)
        return out


class _TimestampLike:
    __slots__ = ("seconds", "nanos")

    def __init__(self, seconds: int, nanos: int) -> None:
        self.seconds = seconds
        self.nanos = nanos


def _parse_timestamp(data: bytes) -> Optional[_TimestampLike]:
    seconds = 0
    nanos = 0
    pos = 0
    while pos < len(data):
        tag_varint, pos = _decode_varint(data, pos)
        field_number = tag_varint >> 3
        wire_type = tag_varint & 0x7
        if wire_type == 0:
            value, pos = _decode_varint(data, pos)
            if field_number == 1:
                seconds = value
            elif field_number == 2:
                nanos = value
        else:
            break
    return _TimestampLike(seconds, nanos)


class _InboxMessageProto:
    """Minimal hand-rolled protobuf decoder for InboxMessage.

    Proto field mapping (from ambient/v1/inbox.proto):
      1: id          (string, wire 2)
      2: agent_id    (string, wire 2)
      3: from_agent_id (optional string, wire 2)
      4: from_name   (optional string, wire 2)
      5: body        (string, wire 2)
      6: read        (optional bool, wire 0)
      7: created_at  (Timestamp, wire 2)
      8: updated_at  (Timestamp, wire 2)
    """

    __slots__ = (
        "id",
        "agent_id",
        "from_agent_id",
        "from_name",
        "body",
        "read",
        "created_at",
        "updated_at",
    )

    def __init__(self) -> None:
        self.id: str = ""
        self.agent_id: str = ""
        self.from_agent_id: Optional[str] = None
        self.from_name: Optional[str] = None
        self.body: str = ""
        self.read: Optional[bool] = None
        self.created_at: Optional[_TimestampLike] = None
        self.updated_at: Optional[_TimestampLike] = None

    @classmethod
    def FromString(cls, data: bytes) -> _InboxMessageProto:
        msg = cls()
        pos = 0
        while pos < len(data):
            tag_varint, pos = _decode_varint(data, pos)
            field_number = tag_varint >> 3
            wire_type = tag_varint & 0x7
            if wire_type == 2:
                length, pos = _decode_varint(data, pos)
                value_bytes = data[pos : pos + length]
                pos += length
                if field_number == 1:
                    msg.id = value_bytes.decode("utf-8", errors="replace")
                elif field_number == 2:
                    msg.agent_id = value_bytes.decode("utf-8", errors="replace")
                elif field_number == 3:
                    msg.from_agent_id = value_bytes.decode("utf-8", errors="replace")
                elif field_number == 4:
                    msg.from_name = value_bytes.decode("utf-8", errors="replace")
                elif field_number == 5:
                    msg.body = value_bytes.decode("utf-8", errors="replace")
                elif field_number == 7:
                    msg.created_at = _parse_timestamp(value_bytes)
                elif field_number == 8:
                    msg.updated_at = _parse_timestamp(value_bytes)
            elif wire_type == 0:
                value, pos = _decode_varint(data, pos)
                if field_number == 6:
                    msg.read = bool(value)
            else:
                break
        return msg
