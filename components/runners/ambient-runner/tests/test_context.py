"""Tests for RunnerContext."""

import os

import pytest

from ambient_runner.platform.context import RunnerContext


class TestRunnerContextGetEnv:
    """Verify get_env() reads live from os.environ and respects explicit overrides."""

    def test_get_env_sees_os_environ_mutations_after_creation(self):
        """get_env() must return current os.environ value when key is mutated at runtime."""
        key = "_RUNNER_CONTEXT_TEST_MUTATION_"
        try:
            if key in os.environ:
                del os.environ[key]
            ctx = RunnerContext(session_id="s1", workspace_path="/tmp")
            assert ctx.get_env(key) is None
            os.environ[key] = "runtime-mutated"
            assert ctx.get_env(key) == "runtime-mutated"
        finally:
            os.environ.pop(key, None)

    def test_explicit_overrides_win_over_os_environ(self):
        """Explicit overrides passed at construction must take precedence over os.environ."""
        key = "_RUNNER_CONTEXT_TEST_OVERRIDE_"
        try:
            os.environ[key] = "from-os-environ"
            ctx = RunnerContext(
                session_id="s1",
                workspace_path="/tmp",
                environment={key: "from-constructor"},
            )
            assert ctx.get_env(key) == "from-constructor"
            os.environ[key] = "mutated-after-creation"
            assert ctx.get_env(key) == "from-constructor"
        finally:
            os.environ.pop(key, None)
