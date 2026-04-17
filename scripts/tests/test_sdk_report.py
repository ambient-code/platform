"""Tests for the SDK feature analysis report generator."""

import sys
import textwrap
from pathlib import Path


sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from sdk_report import (
    ActionRequired,
    Feature,
    FeatureReport,
    FeatureStatus,
    RunnerRelevance,
    build_report,
    parse_release_body,
    render_report_markdown,
)


class TestParseReleaseBody:
    def test_parses_claude_sdk_format(self):
        body = textwrap.dedent("""\
            ### New Features

            - **Session info**: Added `tag` and `created_at` fields to `SDKSessionInfo` (#667)

            ### Bug Fixes

            - **MCP init**: Fixed an issue where MCP servers failed to register (#630)

            ### Internal/Other Changes

            - Updated bundled Claude CLI to version 2.1.81
        """)
        features = parse_release_body("0.1.50", body)

        assert len(features) == 3

        session_info = features[0]
        assert session_info.name == "Session info"
        assert session_info.category == "feature"
        assert session_info.version == "0.1.50"
        assert "#667" in session_info.pr_ref

        bugfix = features[1]
        assert bugfix.name == "MCP init"
        assert bugfix.category == "bugfix"

        internal = features[2]
        assert internal.status == FeatureStatus.INTERNAL

    def test_parses_anthropic_sdk_format(self):
        body = textwrap.dedent("""\
            ## 0.86.0 (2026-03-18)

            Full Changelog: [v0.85.0...v0.86.0](https://github.com/anthropics/anthropic-sdk-python/compare/v0.85.0...v0.86.0)

            ### Features

            * add support for filesystem memory tools ([#1247](https://github.com/anthropics/anthropic-sdk-python/issues/1247)) ([235d218](https://github.com/anthropics/anthropic-sdk-python/commit/235d218))
            * **api:** manual updates ([86dbe4a](https://github.com/anthropics/anthropic-sdk-python/commit/86dbe4a))

            ### Bug Fixes

            * **client:** add missing 413 error handler ([#1554](https://github.com/anthropics/anthropic-sdk-python/issues/1554)) ([abc1234](https://github.com/anthropics/anthropic-sdk-python/commit/abc1234))
        """)
        features = parse_release_body("0.86.0", body)

        # Should have 3 features (filesystem memory, manual updates as internal, bug fix)
        assert len(features) >= 2

        # First feature should be filesystem memory tools
        fs_feature = next(f for f in features if "filesystem" in f.name.lower())
        assert fs_feature.category == "feature"
        assert "#1247" in fs_feature.pr_ref

        # Bug fix should be parsed
        bugfix = next(f for f in features if f.category == "bugfix")
        assert "413" in bugfix.description or "client" in bugfix.name.lower()

    def test_merges_sub_bullets(self):
        body = textwrap.dedent("""\
            ### New Features

            - **New hook events**: Added support for three new hook event types (#545):
              - `Notification` — for handling notification events
              - `SubagentStart` — for handling subagent startup
              - `PermissionRequest` — for handling permission requests
        """)
        features = parse_release_body("0.1.29", body)

        # Should produce ONE feature, not four
        assert len(features) == 1
        assert features[0].name == "New hook events"
        assert "Notification" in features[0].description
        assert "SubagentStart" in features[0].description
        assert "PermissionRequest" in features[0].description

    def test_empty_body(self):
        assert parse_release_body("0.1.50", "") == []
        assert parse_release_body("0.1.50", None) == []

    def test_deprecation_detected(self):
        body = textwrap.dedent("""\
            ### New Features

            - **Thinking config**: The `thinking` field takes precedence over the now-deprecated `max_thinking_tokens` field (#565)
        """)
        features = parse_release_body("0.1.36", body)
        assert features[0].status == FeatureStatus.DEPRECATED
        assert features[0].action == ActionRequired.MIGRATE

    def test_internal_noise_filtered(self):
        body = textwrap.dedent("""\
            ### Internal/Other Changes

            - Updated bundled Claude CLI to version 2.1.81
            - Hardened PyPI publish workflow against partial-upload failures (#700)
            - Added daily PyPI storage quota monitoring (#705)
        """)
        features = parse_release_body("0.1.50", body)
        for f in features:
            assert f.status == FeatureStatus.INTERNAL
            assert f.action == ActionRequired.NONE


class TestFeatureDetection:
    def test_opt_in_detected(self):
        body = "### New Features\n\n- **Effort option**: Added `effort` field to `ClaudeAgentOptions` (#565)"
        features = parse_release_body("0.1.36", body)
        assert features[0].action == ActionRequired.OPT_IN
        assert features[0].default_behavior == "Opt-in"

    def test_claude_relevance(self):
        body = "### New Features\n\n- **MCP control**: Added `add_mcp_server()` for MCP management (#620)"
        features = parse_release_body("0.1.46", body)
        assert features[0].runner_relevance == RunnerRelevance.CLAUDE

    def test_beta_status(self):
        body = "### New Features\n\n- **Structured outputs beta**: Added support for structured outputs beta (#73)"
        features = parse_release_body("0.73.0", body)
        assert features[0].status == FeatureStatus.BETA

    def test_ga_status(self):
        body = "### Features\n\n* **api:** GA thinking-display-setting ([207340c](https://example.com/commit/207340c))"
        features = parse_release_body("0.85.0", body)
        ga_feature = next(
            (
                f
                for f in features
                if "ga" in f.name.lower() or "ga" in f.description.lower()
            ),
            None,
        )
        assert ga_feature is not None
        assert ga_feature.status == FeatureStatus.GA


class TestBuildReport:
    def test_builds_report_from_entries(self):
        from dataclasses import dataclass

        @dataclass
        class FakeEntry:
            version: str
            body: str

        entries = [
            FakeEntry(
                "0.1.50",
                "### New Features\n\n- **Session info**: Added tag field (#667)",
            ),
            FakeEntry(
                "0.1.49",
                "### Bug Fixes\n\n- **Streaming**: Fixed streaming issue (#671)",
            ),
        ]

        report = build_report("claude-agent-sdk", "0.1.48", "0.1.50", entries)

        assert report.package_name == "claude-agent-sdk"
        assert report.from_version == "0.1.48"
        assert report.to_version == "0.1.50"
        assert len(report.features) == 2


class TestRenderReportMarkdown:
    def test_renders_feature_table(self):
        report = FeatureReport(
            package_name="claude-agent-sdk",
            from_version="0.1.23",
            to_version="0.1.50",
            features=[
                Feature(
                    name="Session info",
                    version="0.1.50",
                    description="Added tag field",
                    category="feature",
                    default_behavior="Opt-in",
                    status=FeatureStatus.NEW,
                    action=ActionRequired.OPT_IN,
                    runner_relevance=RunnerRelevance.CLAUDE,
                    pr_ref="#667",
                ),
            ],
        )
        md = render_report_markdown([report])
        assert "Session info" in md
        assert "| Yes | - |" in md  # Claude=Yes, Gemini=-
        assert "Opt-in" in md
        assert "## Action Items" in md

    def test_renders_behavior_changes(self):
        report = FeatureReport(
            package_name="claude-agent-sdk",
            from_version="0.1.23",
            to_version="0.1.50",
            features=[
                Feature(
                    name="Thinking config",
                    version="0.1.36",
                    description="now-deprecated max_thinking_tokens",
                    category="behavior_change",
                    default_behavior="Changed",
                    status=FeatureStatus.DEPRECATED,
                    action=ActionRequired.MIGRATE,
                    runner_relevance=RunnerRelevance.CLAUDE,
                ),
            ],
        )
        md = render_report_markdown([report])
        assert "## Behavior Changes" in md
        assert "Thinking config" in md
        assert "Migrate" in md
        assert "MIGRATE" in md  # Action Items section

    def test_internal_not_in_feature_table(self):
        report = FeatureReport(
            package_name="claude-agent-sdk",
            from_version="0.1.23",
            to_version="0.1.24",
            features=[
                Feature(
                    name="Updated bundled CLI",
                    version="0.1.24",
                    description="Updated bundled Claude CLI to 2.1.22",
                    category="internal",
                    default_behavior="N/A",
                    status=FeatureStatus.INTERNAL,
                    action=ActionRequired.NONE,
                    runner_relevance=RunnerRelevance.CLAUDE,
                ),
            ],
        )
        md = render_report_markdown([report])
        assert "New Features" not in md
        assert "Internal changes:" in md

    def test_action_items_section(self):
        report = FeatureReport(
            package_name="claude-agent-sdk",
            from_version="0.1.23",
            to_version="0.1.50",
            features=[
                Feature(
                    name="Effort option",
                    version="0.1.36",
                    description="Added effort field",
                    category="feature",
                    default_behavior="Opt-in",
                    status=FeatureStatus.NEW,
                    action=ActionRequired.OPT_IN,
                    runner_relevance=RunnerRelevance.CLAUDE,
                ),
            ],
        )
        md = render_report_markdown([report])
        assert "## Action Items" in md
        assert "- [ ] **Effort option**" in md

    def test_cli_version_table(self):
        report = FeatureReport(
            package_name="claude-agent-sdk",
            from_version="0.1.23",
            to_version="0.1.24",
            features=[
                Feature(
                    name="Updated bundled CLI",
                    version="0.1.24",
                    description="Updated bundled Claude CLI to version 2.1.22",
                    category="internal",
                    default_behavior="N/A",
                    status=FeatureStatus.INTERNAL,
                    action=ActionRequired.NONE,
                    runner_relevance=RunnerRelevance.CLAUDE,
                ),
            ],
        )
        md = render_report_markdown([report])
        assert "Bundled Claude CLI Versions" in md
        assert "| 0.1.24 | 2.1.22 |" in md

    def test_multi_package_report(self):
        claude_report = FeatureReport(
            package_name="claude-agent-sdk",
            from_version="0.1.23",
            to_version="0.1.50",
            features=[
                Feature(
                    name="Session info",
                    version="0.1.50",
                    description="Added tag field",
                    category="feature",
                    default_behavior="Opt-in",
                    status=FeatureStatus.NEW,
                    action=ActionRequired.OPT_IN,
                    runner_relevance=RunnerRelevance.CLAUDE,
                ),
            ],
        )
        anthropic_report = FeatureReport(
            package_name="anthropic",
            from_version="0.68.0",
            to_version="0.86.0",
            features=[
                Feature(
                    name="Filesystem memory tools",
                    version="0.86.0",
                    description="Added filesystem memory tools",
                    category="feature",
                    default_behavior="Available",
                    status=FeatureStatus.NEW,
                    action=ActionRequired.REVIEW,
                    runner_relevance=RunnerRelevance.CLAUDE,
                ),
            ],
        )
        md = render_report_markdown([claude_report, anthropic_report])
        assert "claude-agent-sdk" in md
        assert "anthropic" in md
        assert "Session info" in md
        assert "Filesystem memory tools" in md

    def test_tldr_section(self):
        report = FeatureReport(
            package_name="claude-agent-sdk",
            from_version="0.1.23",
            to_version="0.1.50",
            features=[
                Feature(
                    name="Effort option",
                    version="0.1.36",
                    description="Added effort field",
                    category="feature",
                    default_behavior="Opt-in",
                    status=FeatureStatus.NEW,
                    action=ActionRequired.OPT_IN,
                    runner_relevance=RunnerRelevance.CLAUDE,
                ),
                Feature(
                    name="Streaming fix",
                    version="0.1.35",
                    description="Fixed streaming",
                    category="bugfix",
                    default_behavior="N/A",
                    status=FeatureStatus.NEW,
                    action=ActionRequired.NONE,
                    runner_relevance=RunnerRelevance.CLAUDE,
                ),
            ],
        )
        md = render_report_markdown([report])
        assert "### TL;DR" in md
        assert "1 opt-in feature(s)" in md
        assert "1 bug fix(es)" in md
