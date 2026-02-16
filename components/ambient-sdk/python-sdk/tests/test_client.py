import os

import pytest

from ambient_platform.client import AmbientClient


class TestClientValidation:
    def test_empty_token_raises(self):
        with pytest.raises(ValueError, match="token cannot be empty"):
            AmbientClient(base_url="http://localhost:8080", token="", project="test")

    def test_short_token_raises(self):
        with pytest.raises(ValueError, match="too short"):
            AmbientClient(base_url="http://localhost:8080", token="abc", project="test")

    def test_placeholder_token_raises(self):
        with pytest.raises(ValueError, match="placeholder"):
            AmbientClient(
                base_url="http://localhost:8080",
                token="YOUR_TOKEN_HERE",
                project="test",
            )

    def test_empty_project_raises(self):
        with pytest.raises(ValueError, match="project cannot be empty"):
            AmbientClient(
                base_url="http://localhost:8080",
                token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
                project="",
            )

    def test_invalid_project_chars_raises(self):
        with pytest.raises(ValueError, match="alphanumeric"):
            AmbientClient(
                base_url="http://localhost:8080",
                token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
                project="bad project!",
            )

    def test_long_project_raises(self):
        with pytest.raises(ValueError, match="63 characters"):
            AmbientClient(
                base_url="http://localhost:8080",
                token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
                project="a" * 64,
            )

    def test_empty_base_url_raises(self):
        with pytest.raises(ValueError, match="base URL cannot be empty"):
            AmbientClient(
                base_url="",
                token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
                project="test",
            )

    def test_invalid_url_scheme_raises(self):
        with pytest.raises(ValueError, match="scheme must be http or https"):
            AmbientClient(
                base_url="ftp://api.test.com",
                token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
                project="test",
            )

    def test_placeholder_domain_raises(self):
        with pytest.raises(ValueError, match="placeholder domain"):
            AmbientClient(
                base_url="https://example.com",
                token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
                project="test",
            )

    def test_valid_client_creation(self):
        client = AmbientClient(
            base_url="https://api.real-platform.com",
            token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
            project="my-project",
        )
        assert client._base_url == "https://api.real-platform.com"
        assert client._project == "my-project"
        assert client._base_path == "/api/ambient-api-server/v1"
        client.close()

    def test_trailing_slash_stripped(self):
        client = AmbientClient(
            base_url="https://api.real-platform.com/",
            token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
            project="my-project",
        )
        assert client._base_url == "https://api.real-platform.com"
        client.close()

    def test_context_manager(self):
        with AmbientClient(
            base_url="https://api.real-platform.com",
            token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
            project="my-project",
        ) as client:
            assert client._project == "my-project"


class TestClientFromEnv:
    def test_missing_token_raises(self, monkeypatch):
        monkeypatch.delenv("AMBIENT_TOKEN", raising=False)
        monkeypatch.delenv("AMBIENT_PROJECT", raising=False)
        with pytest.raises(ValueError, match="AMBIENT_TOKEN"):
            AmbientClient.from_env()

    def test_missing_project_raises(self, monkeypatch):
        monkeypatch.setenv("AMBIENT_TOKEN", "sha256~abcdefghijklmnopqrstuvwxyz1234567890")
        monkeypatch.delenv("AMBIENT_PROJECT", raising=False)
        with pytest.raises(ValueError, match="AMBIENT_PROJECT"):
            AmbientClient.from_env()

    def test_defaults_url(self, monkeypatch):
        monkeypatch.setenv("AMBIENT_TOKEN", "sha256~abcdefghijklmnopqrstuvwxyz1234567890")
        monkeypatch.setenv("AMBIENT_PROJECT", "test-project")
        monkeypatch.delenv("AMBIENT_API_URL", raising=False)
        client = AmbientClient.from_env()
        assert client._base_url == "http://localhost:8080"
        client.close()

    def test_custom_url(self, monkeypatch):
        monkeypatch.setenv("AMBIENT_TOKEN", "sha256~abcdefghijklmnopqrstuvwxyz1234567890")
        monkeypatch.setenv("AMBIENT_PROJECT", "test-project")
        monkeypatch.setenv("AMBIENT_API_URL", "https://custom.api.com")
        client = AmbientClient.from_env()
        assert client._base_url == "https://custom.api.com"
        client.close()


class TestClientResourceAccessors:
    def setup_method(self):
        self.client = AmbientClient(
            base_url="https://api.real-platform.com",
            token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
            project="test-project",
        )

    def teardown_method(self):
        self.client.close()

    def test_sessions_accessor(self):
        api = self.client.sessions
        assert api is not None
        assert self.client.sessions is api

    def test_projects_accessor(self):
        assert self.client.projects is not None

    def test_project_settingss_accessor(self):
        assert self.client.project_settingss is not None

    def test_users_accessor(self):
        assert self.client.users is not None

    def test_api_caching(self):
        api1 = self.client.sessions
        api2 = self.client.sessions
        assert api1 is api2


class TestSessionAPIWP6Methods:
    def setup_method(self):
        self.client = AmbientClient(
            base_url="https://api.real-platform.com",
            token="sha256~abcdefghijklmnopqrstuvwxyz1234567890",
            project="test-project",
        )

    def teardown_method(self):
        self.client.close()

    def test_has_start_method(self):
        assert hasattr(self.client.sessions, "start")
        assert callable(self.client.sessions.start)

    def test_has_stop_method(self):
        assert hasattr(self.client.sessions, "stop")
        assert callable(self.client.sessions.stop)

    def test_has_update_status_method(self):
        assert hasattr(self.client.sessions, "update_status")
        assert callable(self.client.sessions.update_status)

    def test_has_all_crud_and_action_methods(self):
        api = self.client.sessions
        expected = ["create", "get", "list", "update", "update_status", "start", "stop", "list_all"]
        for method_name in expected:
            assert hasattr(api, method_name), f"SessionAPI missing method: {method_name}"
            assert callable(getattr(api, method_name)), f"SessionAPI.{method_name} is not callable"

    def test_non_session_api_has_no_actions(self):
        api = self.client.users
        assert not hasattr(api, "start")
        assert not hasattr(api, "stop")
        assert not hasattr(api, "update_status")
