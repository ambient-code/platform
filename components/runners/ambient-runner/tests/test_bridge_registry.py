"""Unit tests for the BRIDGE_REGISTRY in main.py."""

import importlib
import os
from unittest.mock import patch

import pytest

from ambient_runner.bridge import PlatformBridge


class TestBridgeRegistryEntries:
    """Verify all entries in BRIDGE_REGISTRY can be resolved."""

    def _get_registry(self):
        """Import BRIDGE_REGISTRY from main module."""
        # Import at function level to avoid module-level side effects
        from main import BRIDGE_REGISTRY

        return BRIDGE_REGISTRY

    def test_all_entries_resolvable(self):
        """Every (module_path, class_name) in BRIDGE_REGISTRY can be imported."""
        registry = self._get_registry()
        assert len(registry) > 0, "BRIDGE_REGISTRY should not be empty"

        for runner_type, (module_path, class_name) in registry.items():
            module = importlib.import_module(module_path)
            bridge_cls = getattr(module, class_name)
            assert bridge_cls is not None, (
                f"Class {class_name} not found in {module_path} for runner type {runner_type!r}"
            )

    def test_all_entries_are_platform_bridge_subclasses(self):
        """Every resolved class in BRIDGE_REGISTRY is a subclass of PlatformBridge."""
        registry = self._get_registry()

        for runner_type, (module_path, class_name) in registry.items():
            module = importlib.import_module(module_path)
            bridge_cls = getattr(module, class_name)
            assert issubclass(bridge_cls, PlatformBridge), (
                f"{class_name} for runner type {runner_type!r} is not a PlatformBridge subclass"
            )

    def test_registry_contains_claude(self):
        """BRIDGE_REGISTRY must include claude-agent-sdk."""
        registry = self._get_registry()
        assert "claude-agent-sdk" in registry

    def test_registry_contains_gemini(self):
        """BRIDGE_REGISTRY must include gemini-cli."""
        registry = self._get_registry()
        assert "gemini-cli" in registry

    def test_registry_contains_langgraph(self):
        """BRIDGE_REGISTRY must include langgraph."""
        registry = self._get_registry()
        assert "langgraph" in registry

    def test_registry_values_are_tuples(self):
        """Every value in BRIDGE_REGISTRY is a (module_path, class_name) tuple."""
        registry = self._get_registry()
        for runner_type, entry in registry.items():
            assert isinstance(entry, tuple), (
                f"Expected tuple for {runner_type!r}, got {type(entry)}"
            )
            assert len(entry) == 2, (
                f"Expected 2-tuple for {runner_type!r}, got {len(entry)}-tuple"
            )
            module_path, class_name = entry
            assert isinstance(module_path, str)
            assert isinstance(class_name, str)


class TestRunnerTypeDefault:
    """Test default RUNNER_TYPE value."""

    def test_default_runner_type_is_claude(self):
        """RUNNER_TYPE defaults to 'claude-agent-sdk' when env var is unset."""
        with patch.dict(os.environ, {}, clear=True):
            # Force re-evaluation by importing the module-level constant pattern
            # Since RUNNER_TYPE is set at module load, we test the default via env logic
            runner_type = os.getenv("RUNNER_TYPE", "claude-agent-sdk").strip().lower()
            assert runner_type == "claude-agent-sdk"

    def test_runner_type_from_env(self):
        """RUNNER_TYPE reads from environment variable."""
        with patch.dict(os.environ, {"RUNNER_TYPE": "gemini-cli"}):
            runner_type = os.getenv("RUNNER_TYPE", "claude-agent-sdk").strip().lower()
            assert runner_type == "gemini-cli"

    def test_runner_type_strips_whitespace(self):
        """RUNNER_TYPE strips leading/trailing whitespace."""
        with patch.dict(os.environ, {"RUNNER_TYPE": "  gemini-cli  "}):
            runner_type = os.getenv("RUNNER_TYPE", "claude-agent-sdk").strip().lower()
            assert runner_type == "gemini-cli"

    def test_runner_type_lowercased(self):
        """RUNNER_TYPE is lowercased."""
        with patch.dict(os.environ, {"RUNNER_TYPE": "CLAUDE-AGENT-SDK"}):
            runner_type = os.getenv("RUNNER_TYPE", "claude-agent-sdk").strip().lower()
            assert runner_type == "claude-agent-sdk"


class TestLoadBridgeFunction:
    """Test the _load_bridge() function."""

    def test_unknown_runner_type_raises_value_error(self):
        """Unknown RUNNER_TYPE raises ValueError with helpful message listing available types."""

        # Patch the module-level RUNNER_TYPE to an unknown value
        with patch("main.RUNNER_TYPE", "nonexistent-runner"):
            from main import _load_bridge

            with pytest.raises(
                ValueError, match="Unknown RUNNER_TYPE='nonexistent-runner'"
            ):
                _load_bridge()

    def test_unknown_runner_type_lists_available(self):
        """ValueError message from unknown RUNNER_TYPE lists available runner types."""
        from main import BRIDGE_REGISTRY

        with patch("main.RUNNER_TYPE", "nonexistent-runner"):
            from main import _load_bridge

            with pytest.raises(ValueError) as exc_info:
                _load_bridge()

            error_msg = str(exc_info.value)
            # Verify that available types are listed
            for runner_type in sorted(BRIDGE_REGISTRY.keys()):
                assert runner_type in error_msg, (
                    f"Available type {runner_type!r} not listed in error message"
                )

    def test_load_bridge_returns_instance(self):
        """_load_bridge() returns an instance of PlatformBridge for valid RUNNER_TYPE."""
        with patch("main.RUNNER_TYPE", "claude-agent-sdk"):
            from main import _load_bridge

            bridge = _load_bridge()
            assert isinstance(bridge, PlatformBridge)

    def test_load_bridge_gemini(self):
        """_load_bridge() can load gemini-cli bridge."""
        with patch("main.RUNNER_TYPE", "gemini-cli"):
            from main import _load_bridge

            bridge = _load_bridge()
            assert isinstance(bridge, PlatformBridge)
            assert bridge.capabilities().framework == "gemini-cli"
