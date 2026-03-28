from __future__ import annotations

import logging
import os
from pathlib import Path
from typing import Optional

import grpc

logger = logging.getLogger(__name__)

_ENV_GRPC_URL = "AMBIENT_GRPC_URL"
_ENV_TOKEN = "BOT_TOKEN"
_ENV_USE_TLS = "AMBIENT_GRPC_USE_TLS"
_ENV_CA_CERT = "AMBIENT_GRPC_CA_CERT_FILE"
_DEFAULT_GRPC_URL = "ambient-api-server:9000"
_SERVICE_CA_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
_BOT_TOKEN_FILE = Path("/var/run/secrets/ambient/bot-token")


def _read_current_token() -> str:
    """Read the bot token, preferring the kubelet-rotated file mount over env var."""
    try:
        if _BOT_TOKEN_FILE.exists():
            token = _BOT_TOKEN_FILE.read_text().strip()
            if token:
                return token
    except OSError:
        pass
    return os.environ.get(_ENV_TOKEN, "")


def _load_ca_cert(ca_cert_file: Optional[str]) -> Optional[bytes]:
    """Load CA cert from explicit path, then service-ca fallback, then None."""
    candidates = [ca_cert_file, _SERVICE_CA_PATH]
    for path in candidates:
        if path and os.path.exists(path):
            try:
                with open(path, "rb") as f:
                    return f.read()
            except OSError:
                pass
    return None


def _build_channel(
    grpc_url: str, token: str, use_tls: bool = False, ca_cert_file: Optional[str] = None
) -> grpc.Channel:
    """Build a gRPC channel with optional TLS and bearer token call credentials."""
    logger.info(
        "[GRPC CHANNEL] Building channel: url=%s tls=%s token_present=%s ca_cert=%s",
        grpc_url,
        use_tls,
        bool(token),
        ca_cert_file,
    )
    if use_tls:
        call_creds = grpc.access_token_call_credentials(token) if token else None
        ca_cert = _load_ca_cert(ca_cert_file)
        channel_creds = grpc.ssl_channel_credentials(root_certificates=ca_cert)
        if call_creds:
            logger.info("[GRPC CHANNEL] Using TLS + bearer token credentials")
            return grpc.secure_channel(
                grpc_url, grpc.composite_channel_credentials(channel_creds, call_creds)
            )
        logger.info("[GRPC CHANNEL] Using TLS-only credentials (no token)")
        return grpc.secure_channel(grpc_url, channel_creds)
    logger.info("[GRPC CHANNEL] Using insecure channel (no TLS)")
    return grpc.insecure_channel(grpc_url)


class AmbientGRPCClient:
    """gRPC client for the Ambient Platform internal API.

    Intended for use inside runner Job pods where BOT_TOKEN and
    AMBIENT_GRPC_URL are injected by the operator.
    """

    def __init__(
        self,
        grpc_url: str,
        token: str,
        use_tls: bool = False,
        ca_cert_file: Optional[str] = None,
    ) -> None:
        self._grpc_url = grpc_url
        self._token = token
        self._use_tls = use_tls
        self._ca_cert_file = ca_cert_file
        self._channel: Optional[grpc.Channel] = None
        self._session_messages: Optional["SessionMessagesAPI"] = None  # noqa: F821

    @classmethod
    def from_env(cls) -> AmbientGRPCClient:
        """Create client from environment variables."""
        grpc_url = os.environ.get(_ENV_GRPC_URL, _DEFAULT_GRPC_URL)
        token = _read_current_token()
        use_tls = os.environ.get(_ENV_USE_TLS, "").lower() in ("true", "1", "yes")
        ca_cert_file = os.environ.get(_ENV_CA_CERT)
        logger.info(
            "[GRPC CLIENT] Initializing from env: url=%s tls=%s token_len=%d",
            grpc_url,
            use_tls,
            len(token),
        )
        return cls(
            grpc_url=grpc_url, token=token, use_tls=use_tls, ca_cert_file=ca_cert_file
        )

    def reconnect(self) -> None:
        """Close the existing channel and rebuild with a fresh token from the file mount."""
        fresh_token = _read_current_token()
        logger.info(
            "[GRPC CLIENT] Reconnecting with fresh token (len=%d)", len(fresh_token)
        )
        self.close()
        self._token = fresh_token

    def _get_channel(self) -> grpc.Channel:
        if self._channel is None:
            logger.info("[GRPC CHANNEL] Creating new channel to %s", self._grpc_url)
            self._channel = _build_channel(
                self._grpc_url, self._token, self._use_tls, self._ca_cert_file
            )
            logger.info("[GRPC CHANNEL] Channel created successfully")
        return self._channel

    @property
    def session_messages(self) -> "SessionMessagesAPI":  # noqa: F821
        if self._session_messages is None:
            logger.info("[GRPC CLIENT] Creating SessionMessagesAPI stub")
            from ._session_messages_api import SessionMessagesAPI

            self._session_messages = SessionMessagesAPI(
                self._get_channel(), token=self._token, grpc_client=self
            )
            logger.info("[GRPC CLIENT] SessionMessagesAPI ready")
        return self._session_messages

    def close(self) -> None:
        if self._channel is not None:
            self._channel.close()
            self._channel = None
            self._session_messages = None

    def __enter__(self) -> AmbientGRPCClient:
        return self

    def __exit__(self, *args: object) -> None:
        self.close()
