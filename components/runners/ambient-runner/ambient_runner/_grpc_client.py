from __future__ import annotations

import base64
import json
import logging
import os
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Optional

import grpc
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding

from ambient_runner.platform.utils import set_bot_token

logger = logging.getLogger(__name__)

_ENV_GRPC_URL = "AMBIENT_GRPC_URL"
_ENV_TOKEN = "BOT_TOKEN"
_ENV_CP_TOKEN_URL = "AMBIENT_CP_TOKEN_URL"
_ENV_CP_TOKEN_PUBLIC_KEY = "AMBIENT_CP_TOKEN_PUBLIC_KEY"
_ENV_SESSION_ID = "SESSION_ID"
_ENV_USE_TLS = "AMBIENT_GRPC_USE_TLS"
_ENV_CA_CERT = "AMBIENT_GRPC_CA_CERT_FILE"
_DEFAULT_GRPC_URL = "ambient-api-server:9000"
_SERVICE_CA_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
_SA_TOKEN_FILE = Path("/var/run/secrets/kubernetes.io/serviceaccount/token")


_CP_TOKEN_FETCH_ATTEMPTS = 3
_CP_TOKEN_FETCH_TIMEOUT = 10


def _encrypt_session_id(public_key_pem: str, session_id: str) -> str:
    """RSA-OAEP encrypt session_id with the CP public key, return base64-encoded ciphertext."""
    public_key = serialization.load_pem_public_key(public_key_pem.encode())
    ciphertext = public_key.encrypt(
        session_id.encode(),
        padding.OAEP(
            mgf=padding.MGF1(algorithm=hashes.SHA256()),
            algorithm=hashes.SHA256(),
            label=None,
        ),
    )
    return base64.b64encode(ciphertext).decode()


def _validate_cp_token_url(url: str) -> None:
    """Reject non-http(s) or credential-bearing URLs to prevent exfiltration."""
    parsed = urllib.parse.urlparse(url)
    if (
        parsed.scheme not in {"http", "https"}
        or not parsed.netloc
        or parsed.username is not None
        or parsed.password is not None
    ):
        raise RuntimeError(
            f"invalid CP token URL (must be http/https with no credentials): {url!r}"
        )


def _fetch_token_from_cp(
    cp_token_url: str, public_key_pem: str, session_id: str
) -> str:
    """Fetch a fresh API token from the CP /token endpoint.

    Encrypts the session ID with the CP public key and sends it as a Bearer token.
    Retries up to _CP_TOKEN_FETCH_ATTEMPTS times with exponential backoff.
    """
    _validate_cp_token_url(cp_token_url)

    bearer = _encrypt_session_id(public_key_pem, session_id)

    last_err: Exception = RuntimeError("no attempts made")
    for attempt in range(_CP_TOKEN_FETCH_ATTEMPTS):
        if attempt > 0:
            backoff = 2 ** (attempt - 1)
            logger.warning(
                "[GRPC CLIENT] CP token fetch attempt %d/%d failed, retrying in %ds: %s",
                attempt,
                _CP_TOKEN_FETCH_ATTEMPTS,
                backoff,
                last_err,
            )
            time.sleep(backoff)
        try:
            req = urllib.request.Request(
                cp_token_url,
                headers={"Authorization": f"Bearer {bearer}"},
            )
            with urllib.request.urlopen(req, timeout=_CP_TOKEN_FETCH_TIMEOUT) as resp:
                body = json.loads(resp.read())
            token = body.get("token", "")
            if not token:
                raise RuntimeError("CP /token response missing 'token' field")
            logger.info("[GRPC CLIENT] Fetched fresh API token from CP token endpoint")
            set_bot_token(token)
            return token
        except urllib.error.HTTPError as e:
            resp_body = ""
            try:
                resp_body = e.read().decode(errors="replace")
            except Exception:
                pass
            last_err = RuntimeError(f"CP /token HTTP {e.code}: {resp_body}")
        except Exception as e:
            last_err = e

    raise RuntimeError(
        f"CP token endpoint unreachable after {_CP_TOKEN_FETCH_ATTEMPTS} attempts: {last_err}"
    ) from last_err


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
        cp_token_url: str = "",
    ) -> None:
        self._grpc_url = grpc_url
        self._token = token
        self._use_tls = use_tls
        self._ca_cert_file = ca_cert_file
        self._cp_token_url = cp_token_url
        self._channel: Optional[grpc.Channel] = None
        self._session_messages: Optional["SessionMessagesAPI"] = None  # noqa: F821

    @classmethod
    def from_env(cls) -> AmbientGRPCClient:
        """Create client from environment variables."""
        grpc_url = os.environ.get(_ENV_GRPC_URL, _DEFAULT_GRPC_URL)
        cp_token_url = os.environ.get(_ENV_CP_TOKEN_URL, "")
        use_tls = os.environ.get(_ENV_USE_TLS, "").lower() in ("true", "1", "yes")
        ca_cert_file = os.environ.get(_ENV_CA_CERT)
        if cp_token_url:
            public_key_pem = os.environ.get(_ENV_CP_TOKEN_PUBLIC_KEY, "")
            session_id = os.environ.get(_ENV_SESSION_ID, "")
            if not public_key_pem:
                raise RuntimeError(
                    "AMBIENT_CP_TOKEN_PUBLIC_KEY env var is required when AMBIENT_CP_TOKEN_URL is set"
                )
            if not session_id:
                raise RuntimeError(
                    "SESSION_ID env var is required when AMBIENT_CP_TOKEN_URL is set"
                )
            logger.info(
                "[GRPC CLIENT] Fetching token from CP endpoint: url=%s", cp_token_url
            )
            token = _fetch_token_from_cp(cp_token_url, public_key_pem, session_id)
        else:
            token = os.environ.get(_ENV_TOKEN, "")
            logger.info("[GRPC CLIENT] Using BOT_TOKEN env var (local dev mode)")
        logger.info(
            "[GRPC CLIENT] Initializing from env: url=%s tls=%s token_len=%d",
            grpc_url,
            use_tls,
            len(token),
        )
        return cls(
            grpc_url=grpc_url,
            token=token,
            use_tls=use_tls,
            ca_cert_file=ca_cert_file,
            cp_token_url=cp_token_url,
        )

    def reconnect(self) -> None:
        """Close the existing channel and rebuild with a fresh token from the CP endpoint."""
        if self._cp_token_url:
            public_key_pem = os.environ.get(_ENV_CP_TOKEN_PUBLIC_KEY, "")
            session_id = os.environ.get(_ENV_SESSION_ID, "")
            fresh_token = _fetch_token_from_cp(
                self._cp_token_url, public_key_pem, session_id
            )
        else:
            fresh_token = os.environ.get(_ENV_TOKEN, "")
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
