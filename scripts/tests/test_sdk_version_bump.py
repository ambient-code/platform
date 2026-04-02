"""Tests for the SDK version bump script."""

import json
import textwrap
from pathlib import Path
from unittest.mock import MagicMock, patch


# Import the module under test
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
import importlib

sdk_version_bump = importlib.import_module("sdk-version-bump")


class TestVersionTuple:
    def test_simple_version(self):
        assert sdk_version_bump._version_tuple("1.2.3") == (1, 2, 3)

    def test_two_part_version(self):
        assert sdk_version_bump._version_tuple("1.2") == (1, 2)

    def test_pre_release_suffix(self):
        # Pre-release like "1.0.0rc1" should extract leading digits
        assert sdk_version_bump._version_tuple("1.0.0rc1") == (1, 0, 0)
        assert sdk_version_bump._version_tuple("2.1.0b3") == (2, 1, 0)

    def test_empty_string(self):
        assert sdk_version_bump._version_tuple("") == (0,)

    def test_comparison(self):
        assert sdk_version_bump._version_tuple(
            "0.1.50"
        ) > sdk_version_bump._version_tuple("0.1.23")
        assert sdk_version_bump._version_tuple(
            "1.0.0"
        ) > sdk_version_bump._version_tuple("0.99.99")
        assert sdk_version_bump._version_tuple(
            "0.1.23"
        ) == sdk_version_bump._version_tuple("0.1.23")


class TestParseCurrentVersion:
    def test_parses_simple_dependency(self, tmp_path):
        pyproject = tmp_path / "pyproject.toml"
        pyproject.write_text(
            textwrap.dedent("""\
            [project.optional-dependencies]
            claude = [
              "anthropic[vertex]>=0.68.0",
              "claude-agent-sdk>=0.1.23",
            ]
        """)
        )
        assert (
            sdk_version_bump.parse_current_version(pyproject, "claude-agent-sdk")
            == "0.1.23"
        )

    def test_parses_extras_dependency(self, tmp_path):
        pyproject = tmp_path / "pyproject.toml"
        pyproject.write_text(
            textwrap.dedent("""\
            [project.optional-dependencies]
            claude = [
              "anthropic[vertex]>=0.68.0",
              "claude-agent-sdk>=0.1.23",
            ]
        """)
        )
        assert (
            sdk_version_bump.parse_current_version(pyproject, "anthropic[vertex]")
            == "0.68.0"
        )

    def test_returns_none_for_missing(self, tmp_path):
        pyproject = tmp_path / "pyproject.toml"
        pyproject.write_text('[project]\nname = "test"\n')
        assert sdk_version_bump.parse_current_version(pyproject, "nonexistent") is None


class TestUpdatePyprojectVersion:
    def test_updates_version(self, tmp_path):
        pyproject = tmp_path / "pyproject.toml"
        pyproject.write_text(
            textwrap.dedent("""\
            [project.optional-dependencies]
            claude = [
              "anthropic[vertex]>=0.68.0",
              "claude-agent-sdk>=0.1.23",
            ]
        """)
        )
        result = sdk_version_bump.update_pyproject_version(
            pyproject, "claude-agent-sdk", "0.1.23", "0.1.50"
        )
        assert result is True
        content = pyproject.read_text()
        assert '"claude-agent-sdk>=0.1.50"' in content
        # Ensure other deps are untouched
        assert '"anthropic[vertex]>=0.68.0"' in content

    def test_updates_extras_version(self, tmp_path):
        pyproject = tmp_path / "pyproject.toml"
        pyproject.write_text(
            textwrap.dedent("""\
            [project.optional-dependencies]
            claude = [
              "anthropic[vertex]>=0.68.0",
              "claude-agent-sdk>=0.1.23",
            ]
        """)
        )
        result = sdk_version_bump.update_pyproject_version(
            pyproject, "anthropic[vertex]", "0.68.0", "0.86.0"
        )
        assert result is True
        content = pyproject.read_text()
        assert '"anthropic[vertex]>=0.86.0"' in content
        assert '"claude-agent-sdk>=0.1.23"' in content

    def test_returns_false_if_not_found(self, tmp_path):
        pyproject = tmp_path / "pyproject.toml"
        pyproject.write_text('[project]\nname = "test"\n')
        result = sdk_version_bump.update_pyproject_version(
            pyproject, "nonexistent", "1.0.0", "2.0.0"
        )
        assert result is False


class TestFetchPypiLatest:
    def test_successful_fetch(self):
        mock_response = json.dumps({"info": {"version": "0.1.50"}}).encode()
        mock_ctx = MagicMock()
        mock_ctx.__enter__ = MagicMock(
            return_value=MagicMock(read=MagicMock(return_value=mock_response))
        )
        mock_ctx.__exit__ = MagicMock(return_value=False)

        with patch("urllib.request.urlopen", return_value=mock_ctx):
            result = sdk_version_bump.fetch_pypi_latest("claude-agent-sdk")
            assert result == "0.1.50"

    def test_network_failure_returns_none(self):
        with patch("urllib.request.urlopen", side_effect=Exception("timeout")):
            result = sdk_version_bump.fetch_pypi_latest("claude-agent-sdk")
            assert result is None


class TestFetchGithubChangelog:
    def _make_releases(self, versions_and_bodies):
        return json.dumps(
            [{"tag_name": f"v{v}", "body": b} for v, b in versions_and_bodies]
        ).encode()

    def test_filters_versions_in_range(self):
        releases = self._make_releases(
            [
                ("0.1.26", "Release 0.1.26 notes"),
                ("0.1.25", "Release 0.1.25 notes"),
                ("0.1.24", "Release 0.1.24 notes"),
                ("0.1.23", "Should be excluded (current)"),
                ("0.1.22", "Should be excluded (older)"),
            ]
        )
        mock_ctx = MagicMock()
        mock_ctx.__enter__ = MagicMock(
            return_value=MagicMock(read=MagicMock(return_value=releases))
        )
        mock_ctx.__exit__ = MagicMock(return_value=False)

        with patch("urllib.request.urlopen", return_value=mock_ctx):
            entries = sdk_version_bump.fetch_github_changelog(
                "anthropics/claude-agent-sdk-python", "0.1.23", "0.1.26"
            )

        assert len(entries) == 3
        # Newest first
        assert entries[0].version == "0.1.26"
        assert entries[1].version == "0.1.25"
        assert entries[2].version == "0.1.24"
        assert "0.1.23" not in [e.version for e in entries]

    def test_empty_on_no_releases(self):
        releases = json.dumps([]).encode()
        mock_ctx = MagicMock()
        mock_ctx.__enter__ = MagicMock(
            return_value=MagicMock(read=MagicMock(return_value=releases))
        )
        mock_ctx.__exit__ = MagicMock(return_value=False)

        with patch("urllib.request.urlopen", return_value=mock_ctx):
            entries = sdk_version_bump.fetch_github_changelog(
                "anthropics/test", "0.1.0", "0.2.0"
            )
        assert entries == []

    def test_network_failure_returns_empty(self):
        with patch("urllib.request.urlopen", side_effect=Exception("fail")):
            entries = sdk_version_bump.fetch_github_changelog(
                "anthropics/test", "0.1.0", "0.2.0"
            )
        assert entries == []


class TestCheckVersions:
    def test_detects_update_needed(self, tmp_path):
        pyproject = (
            tmp_path / "components" / "runners" / "ambient-runner" / "pyproject.toml"
        )
        pyproject.parent.mkdir(parents=True)
        pyproject.write_text(
            textwrap.dedent("""\
            [project.optional-dependencies]
            claude = [
              "anthropic[vertex]>=0.68.0",
              "claude-agent-sdk>=0.1.23",
            ]
        """)
        )

        with patch.object(sdk_version_bump, "fetch_pypi_latest") as mock_pypi:
            mock_pypi.side_effect = lambda name: {
                "claude-agent-sdk": "0.1.50",
                "anthropic": "0.86.0",
            }.get(name)

            results = sdk_version_bump.check_versions(tmp_path)

        assert len(results) == 2
        sdk_result = next(r for r in results if r.pypi_name == "claude-agent-sdk")
        assert sdk_result.needs_update is True
        assert sdk_result.current_version == "0.1.23"
        assert sdk_result.latest_version == "0.1.50"

        anthropic_result = next(r for r in results if r.pypi_name == "anthropic")
        assert anthropic_result.needs_update is True

    def test_no_update_needed(self, tmp_path):
        pyproject = (
            tmp_path / "components" / "runners" / "ambient-runner" / "pyproject.toml"
        )
        pyproject.parent.mkdir(parents=True)
        pyproject.write_text(
            textwrap.dedent("""\
            [project.optional-dependencies]
            claude = [
              "anthropic[vertex]>=0.86.0",
              "claude-agent-sdk>=0.1.50",
            ]
        """)
        )

        with patch.object(sdk_version_bump, "fetch_pypi_latest") as mock_pypi:
            mock_pypi.side_effect = lambda name: {
                "claude-agent-sdk": "0.1.50",
                "anthropic": "0.86.0",
            }.get(name)

            results = sdk_version_bump.check_versions(tmp_path)

        assert all(not r.needs_update for r in results)


class TestBuildPrTitle:
    def test_single_package(self):
        versions = [
            sdk_version_bump.VersionInfo(
                package_name="claude-agent-sdk",
                pypi_name="claude-agent-sdk",
                github_repo="anthropics/claude-agent-sdk-python",
                current_version="0.1.23",
                latest_version="0.1.50",
                needs_update=True,
            ),
        ]
        title = sdk_version_bump.build_pr_title(versions)
        assert title == "deps(runner): bump claude-agent-sdk 0.1.50"

    def test_multiple_packages(self):
        versions = [
            sdk_version_bump.VersionInfo(
                package_name="claude-agent-sdk",
                pypi_name="claude-agent-sdk",
                github_repo="anthropics/claude-agent-sdk-python",
                current_version="0.1.23",
                latest_version="0.1.50",
                needs_update=True,
            ),
            sdk_version_bump.VersionInfo(
                package_name="anthropic[vertex]",
                pypi_name="anthropic",
                github_repo="anthropics/anthropic-sdk-python",
                current_version="0.68.0",
                latest_version="0.86.0",
                needs_update=True,
            ),
        ]
        title = sdk_version_bump.build_pr_title(versions)
        assert "claude-agent-sdk 0.1.50" in title
        assert "anthropic 0.86.0" in title

    def test_skips_non_update_packages(self):
        versions = [
            sdk_version_bump.VersionInfo(
                package_name="claude-agent-sdk",
                pypi_name="claude-agent-sdk",
                github_repo="anthropics/claude-agent-sdk-python",
                current_version="0.1.50",
                latest_version="0.1.50",
                needs_update=False,
            ),
        ]
        title = sdk_version_bump.build_pr_title(versions)
        assert title == "deps(runner): no updates"
