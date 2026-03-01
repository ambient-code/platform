"""
Tests for ProxyAuthMiddleware — backend proxy authentication.

Verifies that the runner rejects direct AG-UI connections that bypass the
backend proxy when RUNNER_PROXY_SECRET is configured, and that it allows
requests that carry the correct Authorization header.
"""


import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from ambient_runner.middleware.proxy_auth import ProxyAuthMiddleware


# ---------------------------------------------------------------------------
# Minimal test app
# ---------------------------------------------------------------------------


def _make_app() -> FastAPI:
    """Create a minimal FastAPI app with ProxyAuthMiddleware and one endpoint."""
    app = FastAPI()
    app.add_middleware(ProxyAuthMiddleware)

    @app.post("/")
    async def run():
        return {"status": "ok"}

    @app.post("/interrupt")
    async def interrupt():
        return {"status": "ok"}

    @app.post("/feedback")
    async def feedback():
        return {"status": "ok"}

    @app.get("/health")
    async def health():
        return {"status": "healthy"}

    @app.get("/capabilities")
    async def capabilities():
        return {"framework": "test"}

    return app


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture()
def client(monkeypatch):
    """Client with RUNNER_PROXY_SECRET not set (passthrough mode)."""
    monkeypatch.delenv("RUNNER_PROXY_SECRET", raising=False)
    with TestClient(_make_app()) as c:
        yield c


@pytest.fixture()
def secure_client(monkeypatch):
    """Client with RUNNER_PROXY_SECRET=test-secret-xyz."""
    monkeypatch.setenv("RUNNER_PROXY_SECRET", "test-secret-xyz")
    with TestClient(_make_app()) as c:
        yield c


# ---------------------------------------------------------------------------
# Passthrough mode (RUNNER_PROXY_SECRET not set)
# ---------------------------------------------------------------------------


class TestPassthroughMode:
    """When RUNNER_PROXY_SECRET is absent all requests pass through."""

    def test_run_allowed_without_auth(self, client):
        resp = client.post("/", json={"messages": []})
        assert resp.status_code == 200

    def test_interrupt_allowed_without_auth(self, client):
        resp = client.post("/interrupt")
        assert resp.status_code == 200

    def test_health_allowed_without_auth(self, client):
        resp = client.get("/health")
        assert resp.status_code == 200

    def test_get_endpoints_allowed_without_auth(self, client):
        resp = client.get("/capabilities")
        assert resp.status_code == 200


# ---------------------------------------------------------------------------
# Enforcement mode (RUNNER_PROXY_SECRET set)
# ---------------------------------------------------------------------------


class TestEnforcementMode:
    """When RUNNER_PROXY_SECRET is set, write endpoints require the correct token."""

    def test_run_rejected_without_auth(self, secure_client):
        resp = secure_client.post("/")
        assert resp.status_code == 403

    def test_run_rejected_with_wrong_token(self, secure_client):
        resp = secure_client.post("/", headers={"Authorization": "Bearer wrong-token"})
        assert resp.status_code == 403

    def test_run_rejected_with_empty_auth(self, secure_client):
        resp = secure_client.post("/", headers={"Authorization": ""})
        assert resp.status_code == 403

    def test_run_allowed_with_correct_token(self, secure_client):
        resp = secure_client.post(
            "/", headers={"Authorization": "Bearer test-secret-xyz"}
        )
        assert resp.status_code == 200

    def test_interrupt_rejected_without_auth(self, secure_client):
        resp = secure_client.post("/interrupt")
        assert resp.status_code == 403

    def test_interrupt_allowed_with_correct_token(self, secure_client):
        resp = secure_client.post(
            "/interrupt", headers={"Authorization": "Bearer test-secret-xyz"}
        )
        assert resp.status_code == 200

    def test_feedback_rejected_without_auth(self, secure_client):
        resp = secure_client.post("/feedback")
        assert resp.status_code == 403

    def test_feedback_allowed_with_correct_token(self, secure_client):
        resp = secure_client.post(
            "/feedback", headers={"Authorization": "Bearer test-secret-xyz"}
        )
        assert resp.status_code == 200

    def test_health_always_public(self, secure_client):
        """Health endpoint must not require auth (K8s liveness probes)."""
        resp = secure_client.get("/health")
        assert resp.status_code == 200

    def test_get_endpoints_allowed_without_auth(self, secure_client):
        """Read-only GET requests do not require auth."""
        resp = secure_client.get("/capabilities")
        assert resp.status_code == 200

    def test_error_response_has_detail_field(self, secure_client):
        """Rejected requests return a JSON body with a 'detail' field."""
        resp = secure_client.post("/")
        assert resp.status_code == 403
        body = resp.json()
        assert "detail" in body
        assert "backend proxy" in body["detail"]

    def test_malformed_auth_scheme_rejected(self, secure_client):
        """Token without 'Bearer ' prefix is rejected."""
        resp = secure_client.post("/", headers={"Authorization": "test-secret-xyz"})
        assert resp.status_code == 403

    def test_basic_auth_rejected(self, secure_client):
        """Basic auth is not a valid scheme for proxy auth."""
        resp = secure_client.post("/", headers={"Authorization": "Basic dGVzdA=="})
        assert resp.status_code == 403
