"""Unit tests for auto-analysis (standalone, no bridge dependency)."""

import asyncio
from unittest.mock import MagicMock, patch

import pytest

INTEL_CLIENT_PATH = "ambient_runner.tools.intelligence_api.IntelligenceAPIClient"
VERTEX_CLIENT_PATH = "ambient_runner.tools.vertex_client.VertexAnthropicClient"


class TestRunAutoAnalysis:
    """Test the run_auto_analysis function."""

    def _run(self, coro):
        return asyncio.run(coro)

    def test_skips_when_intelligence_exists(self):
        from ambient_runner.endpoints.auto_analysis import run_auto_analysis

        mock_client = MagicMock()
        mock_client.intelligence_exists.return_value = True

        with patch(INTEL_CLIENT_PATH, return_value=mock_client):
            self._run(run_auto_analysis("my-repo", "https://github.com/org/repo"))

        # Should not have called Vertex
        # (no assertion on Vertex — it should never be imported)

    def test_skips_when_no_intel_client(self, monkeypatch):
        from ambient_runner.endpoints.auto_analysis import run_auto_analysis

        with patch(INTEL_CLIENT_PATH, side_effect=ValueError("no URL")):
            self._run(run_auto_analysis("my-repo", "https://github.com/org/repo"))

        # Should not crash

    def test_skips_when_no_vertex_client(self):
        from ambient_runner.endpoints.auto_analysis import run_auto_analysis

        mock_intel = MagicMock()
        mock_intel.intelligence_exists.return_value = False

        with patch(INTEL_CLIENT_PATH, return_value=mock_intel), \
             patch(VERTEX_CLIENT_PATH, side_effect=ValueError("no project")):
            self._run(run_auto_analysis("my-repo", "https://github.com/org/repo"))

        # Should not crash

    def test_calls_vertex_and_marks_dirty(self):
        from ambient_runner.endpoints.auto_analysis import run_auto_analysis

        mock_intel = MagicMock()
        mock_intel.intelligence_exists.return_value = False

        mock_vertex = MagicMock()
        mock_vertex.create_message.return_value = {
            "content": [{"type": "text", "text": "Analysis complete."}],
            "stop_reason": "end_turn",
        }

        mock_bridge = MagicMock()

        with patch(INTEL_CLIENT_PATH, return_value=mock_intel), \
             patch(VERTEX_CLIENT_PATH, return_value=mock_vertex):
            self._run(run_auto_analysis("my-repo", "https://github.com/org/repo", mock_bridge))

        mock_vertex.create_message.assert_called_once()
        mock_bridge.mark_dirty.assert_called_once()


class TestAddRepoIntegration:
    """Test that add_repo calls standalone auto-analysis."""

    def test_add_repo_calls_standalone_analysis(self):
        import inspect
        from ambient_runner.endpoints.repos import add_repo

        source = inspect.getsource(add_repo)
        assert "run_auto_analysis" in source
        # Old bridge-dependent function should be gone
        assert "_queue_auto_analysis_if_needed" not in source
