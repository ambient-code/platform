from __future__ import annotations

import base64
import json
import os
from unittest.mock import MagicMock, patch

import pytest
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding, rsa

from ambient_runner._grpc_client import (
    _encrypt_session_id,
    _fetch_token_from_cp,
    _validate_cp_token_url,
)


def generate_keypair():
    private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    private_pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.TraditionalOpenSSL,
        encryption_algorithm=serialization.NoEncryption(),
    ).decode()
    public_pem = (
        private_key.public_key()
        .public_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PublicFormat.SubjectPublicKeyInfo,
        )
        .decode()
    )
    return private_key, private_pem, public_pem


class TestValidateCPTokenURL:
    def test_valid_http(self):
        _validate_cp_token_url("http://ambient-control-plane.svc:8080/token")

    def test_valid_https(self):
        _validate_cp_token_url("https://ambient-control-plane.svc:8080/token")

    def test_rejects_ftp(self):
        with pytest.raises(RuntimeError, match="invalid CP token URL"):
            _validate_cp_token_url("ftp://example.com/token")

    def test_rejects_file(self):
        with pytest.raises(RuntimeError, match="invalid CP token URL"):
            _validate_cp_token_url("file:///etc/passwd")

    def test_rejects_credentials_in_url(self):
        with pytest.raises(RuntimeError, match="invalid CP token URL"):
            _validate_cp_token_url("http://user:pass@example.com/token")

    def test_rejects_empty(self):
        with pytest.raises(RuntimeError, match="invalid CP token URL"):
            _validate_cp_token_url("")

    def test_rejects_no_host(self):
        with pytest.raises(RuntimeError, match="invalid CP token URL"):
            _validate_cp_token_url("http:///token")


class TestEncryptSessionID:
    def test_produces_base64_ciphertext(self):
        _, _, public_pem = generate_keypair()
        result = _encrypt_session_id(public_pem, "my-session-id")
        decoded = base64.b64decode(result)
        assert len(decoded) > 0

    def test_decryptable_with_private_key(self):
        private_key, _, public_pem = generate_keypair()
        session_id = "3BurtLWQNFMLp61XAGFKILYiHoN"

        ciphertext_b64 = _encrypt_session_id(public_pem, session_id)
        ciphertext = base64.b64decode(ciphertext_b64)

        plaintext = private_key.decrypt(
            ciphertext,
            padding.OAEP(
                mgf=padding.MGF1(algorithm=hashes.SHA256()),
                algorithm=hashes.SHA256(),
                label=None,
            ),
        )
        assert plaintext.decode() == session_id

    def test_different_ciphertexts_for_same_input(self):
        _, _, public_pem = generate_keypair()
        result1 = _encrypt_session_id(public_pem, "session-abc")
        result2 = _encrypt_session_id(public_pem, "session-abc")
        assert result1 != result2

    def test_invalid_public_key_raises(self):
        with pytest.raises(Exception):
            _encrypt_session_id("not a pem key", "session-id")


class TestFetchTokenFromCP:
    def _mock_successful_response(self, token: str = "api-token-xyz"):

        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"token": token}).encode()
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)
        return mock_resp

    def test_success(self):
        _, _, public_pem = generate_keypair()
        mock_resp = self._mock_successful_response("test-api-token")

        with patch("urllib.request.urlopen", return_value=mock_resp):
            token = _fetch_token_from_cp(
                "http://cp.svc:8080/token", public_pem, "session-12345678"
            )

        assert token == "test-api-token"

    def test_sends_encrypted_bearer(self):
        _, _, public_pem = generate_keypair()
        mock_resp = self._mock_successful_response()
        captured_req = {}

        def fake_urlopen(req, timeout=None):
            captured_req["req"] = req
            return mock_resp

        with patch("urllib.request.urlopen", side_effect=fake_urlopen):
            _fetch_token_from_cp("http://cp.svc:8080/token", public_pem, "session-abc")

        auth = captured_req["req"].get_header("Authorization")
        assert auth.startswith("Bearer ")
        b64_part = auth[len("Bearer ") :]
        decoded = base64.b64decode(b64_part)
        assert len(decoded) > 0

    def test_retries_on_failure_then_succeeds(self):
        _, _, public_pem = generate_keypair()
        mock_resp = self._mock_successful_response()
        import urllib.error

        call_count = [0]

        def fake_urlopen(req, timeout=None):
            call_count[0] += 1
            if call_count[0] < 3:
                raise urllib.error.URLError("connection refused")
            return mock_resp

        with patch("urllib.request.urlopen", side_effect=fake_urlopen):
            with patch("time.sleep"):
                token = _fetch_token_from_cp(
                    "http://cp.svc:8080/token", public_pem, "session-12345678"
                )

        assert token == "api-token-xyz"
        assert call_count[0] == 3

    def test_raises_after_all_attempts_fail(self):
        _, _, public_pem = generate_keypair()
        import urllib.error

        with patch(
            "urllib.request.urlopen", side_effect=urllib.error.URLError("refused")
        ):
            with patch("time.sleep"):
                with pytest.raises(RuntimeError, match="CP token endpoint unreachable"):
                    _fetch_token_from_cp(
                        "http://cp.svc:8080/token", public_pem, "session-12345678"
                    )

    def test_includes_http_error_body_in_exception(self):
        _, _, public_pem = generate_keypair()
        import urllib.error

        err_body = b"unauthorized: invalid token"
        http_err = urllib.error.HTTPError(
            url="http://cp.svc:8080/token",
            code=401,
            msg="Unauthorized",
            hdrs=None,
            fp=MagicMock(read=MagicMock(return_value=err_body)),
        )

        with patch("urllib.request.urlopen", side_effect=http_err):
            with patch("time.sleep"):
                with pytest.raises(RuntimeError, match="CP /token HTTP 401"):
                    _fetch_token_from_cp(
                        "http://cp.svc:8080/token", public_pem, "session-12345678"
                    )

    def test_raises_on_missing_token_field(self):
        _, _, public_pem = generate_keypair()
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"other": "field"}).encode()
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)

        with patch("urllib.request.urlopen", return_value=mock_resp):
            with patch("time.sleep"):
                with pytest.raises(RuntimeError, match="missing 'token' field"):
                    _fetch_token_from_cp(
                        "http://cp.svc:8080/token", public_pem, "session-12345678"
                    )


class TestSetBotTokenIntegration:
    def test_get_bot_token_returns_cp_fetched_token_after_successful_fetch(self):
        import ambient_runner.platform.utils as utils

        utils._cp_fetched_token = ""

        _, _, public_pem = generate_keypair()
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"token": "oidc-token-for-api-calls"}
        ).encode()
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)

        assert utils.get_bot_token() == "", (
            "get_bot_token() must be empty before any CP fetch"
        )

        with patch("urllib.request.urlopen", return_value=mock_resp):
            _fetch_token_from_cp(
                "http://cp.svc:8080/token", public_pem, "session-12345678"
            )

        assert utils.get_bot_token() == "oidc-token-for-api-calls", (
            "get_bot_token() must return the CP-fetched token so backend API credential "
            "calls are authenticated — regression for HTTP 401 on credential refresh"
        )
        utils._cp_fetched_token = ""

    def test_fetch_from_cp_calls_set_bot_token(self):
        from cryptography.hazmat.primitives.asymmetric import rsa as _rsa

        private_key = _rsa.generate_private_key(public_exponent=65537, key_size=2048)
        public_pem = (
            private_key.public_key()
            .public_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PublicFormat.SubjectPublicKeyInfo,
            )
            .decode()
        )

        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps(
            {"token": "oidc-api-token-abc"}
        ).encode()
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)

        import ambient_runner.platform.utils as utils

        utils._cp_fetched_token = ""

        with patch("urllib.request.urlopen", return_value=mock_resp):
            _fetch_token_from_cp(
                "http://cp.svc:8080/token", public_pem, "session-12345678"
            )

        assert utils.get_bot_token() == "oidc-api-token-abc"
        utils._cp_fetched_token = ""


class TestFromEnvIntegration:
    def test_uses_encrypted_session_id_when_cp_token_url_set(self):
        _, _, public_pem = generate_keypair()
        mock_resp = MagicMock()
        mock_resp.read.return_value = json.dumps({"token": "env-token"}).encode()
        mock_resp.__enter__ = MagicMock(return_value=mock_resp)
        mock_resp.__exit__ = MagicMock(return_value=False)

        env = {
            "AMBIENT_GRPC_URL": "localhost:9000",
            "AMBIENT_CP_TOKEN_URL": "http://cp.svc:8080/token",
            "AMBIENT_CP_TOKEN_PUBLIC_KEY": public_pem,
            "SESSION_ID": "session-test-1234",
            "AMBIENT_GRPC_USE_TLS": "false",
        }

        with patch.dict(os.environ, env, clear=False):
            with patch("urllib.request.urlopen", return_value=mock_resp):
                from ambient_runner._grpc_client import AmbientGRPCClient

                client = AmbientGRPCClient.from_env()

        assert client._token == "env-token"

    def test_falls_back_to_bot_token_when_no_cp_url(self):
        env = {
            "AMBIENT_GRPC_URL": "localhost:9000",
            "BOT_TOKEN": "static-bot-token",
            "AMBIENT_GRPC_USE_TLS": "false",
        }
        env_without_cp = {k: v for k, v in env.items()}

        with patch.dict(os.environ, env_without_cp, clear=False):
            with patch.dict(os.environ, {"AMBIENT_CP_TOKEN_URL": ""}, clear=False):
                from ambient_runner._grpc_client import AmbientGRPCClient

                client = AmbientGRPCClient.from_env()

        assert client._token == "static-bot-token"
