"""Unit tests for model-discovery.py pure functions."""

import importlib.util
import sys
import unittest
from pathlib import Path

# Import model-discovery.py as a module (it has a hyphen in the name)
_spec = importlib.util.spec_from_file_location(
    "model_discovery", Path(__file__).parent / "model-discovery.py"
)
_mod = importlib.util.module_from_spec(_spec)
sys.modules["model_discovery"] = _mod
_spec.loader.exec_module(_mod)

parse_model_family = _mod.parse_model_family
model_id_to_label = _mod.model_id_to_label
keep_latest_versions = _mod.keep_latest_versions


class TestParseModelFamily(unittest.TestCase):
    """Test parse_model_family with both naming conventions."""

    # -- Claude: trailing numeric segments --

    def test_claude_opus(self):
        self.assertEqual(parse_model_family("claude-opus-4-6"), ("claude-opus", (4, 6)))

    def test_claude_sonnet(self):
        self.assertEqual(
            parse_model_family("claude-sonnet-4-5"), ("claude-sonnet", (4, 5))
        )

    def test_claude_haiku(self):
        self.assertEqual(
            parse_model_family("claude-haiku-4-5"), ("claude-haiku", (4, 5))
        )

    # -- Gemini: semver segment --

    def test_gemini_flash(self):
        self.assertEqual(
            parse_model_family("gemini-2.5-flash"), ("gemini-flash", (2, 5))
        )

    def test_gemini_flash_lite(self):
        self.assertEqual(
            parse_model_family("gemini-2.5-flash-lite"), ("gemini-flash-lite", (2, 5))
        )

    def test_gemini_pro(self):
        self.assertEqual(parse_model_family("gemini-2.5-pro"), ("gemini-pro", (2, 5)))

    # -- Qualifier stripping --

    def test_strips_preview(self):
        self.assertEqual(
            parse_model_family("gemini-2.5-flash-preview-04-17"),
            ("gemini-flash", (2, 5)),
        )

    def test_strips_exp_and_date(self):
        self.assertEqual(
            parse_model_family("gemini-2.5-pro-exp-03-25"), ("gemini-pro", (2, 5))
        )

    def test_strips_preview_from_image_model(self):
        self.assertEqual(
            parse_model_family("gemini-3.1-flash-image-preview"),
            ("gemini-flash-image", (3, 1)),
        )

    # -- No version --

    def test_no_version_segments(self):
        self.assertEqual(parse_model_family("some-model"), ("some-model", ()))


class TestModelIdToLabel(unittest.TestCase):
    def test_claude_opus(self):
        self.assertEqual(model_id_to_label("claude-opus-4-6"), "Claude Opus 4.6")

    def test_claude_sonnet(self):
        self.assertEqual(model_id_to_label("claude-sonnet-4-5"), "Claude Sonnet 4.5")

    def test_gemini_flash(self):
        self.assertEqual(model_id_to_label("gemini-2.5-flash"), "Gemini 2.5 Flash")

    def test_gemini_flash_lite(self):
        self.assertEqual(
            model_id_to_label("gemini-2.5-flash-lite"), "Gemini 2.5 Flash Lite"
        )


class TestKeepLatestVersions(unittest.TestCase):
    def test_keeps_latest_two(self):
        models = [
            ("claude-opus-4-1", "anthropic", "anthropic", None),
            ("claude-opus-4-5", "anthropic", "anthropic", None),
            ("claude-opus-4-6", "anthropic", "anthropic", None),
        ]
        result = keep_latest_versions(models, 2)
        ids = [r[0] for r in result]
        self.assertIn("claude-opus-4-6", ids)
        self.assertIn("claude-opus-4-5", ids)
        self.assertNotIn("claude-opus-4-1", ids)

    def test_versionless_always_kept(self):
        models = [
            ("gemini-2.5-flash", "google", "google", None),
            ("some-model", "x", "x", None),
        ]
        result = keep_latest_versions(models, 1)
        ids = [r[0] for r in result]
        # versionless "some-model" has no trailing digits or semver
        self.assertIn("some-model", ids)

    def test_protected_models_exempt(self):
        models = [
            ("claude-opus-4-1", "anthropic", "anthropic", None),
            ("claude-opus-4-5", "anthropic", "anthropic", None),
            ("claude-opus-4-6", "anthropic", "anthropic", None),
        ]
        result = keep_latest_versions(models, 1, protected={"claude-opus-4-1"})
        ids = [r[0] for r in result]
        # 4-1 is protected so kept despite version limit of 1
        self.assertIn("claude-opus-4-1", ids)
        self.assertIn("claude-opus-4-6", ids)
        self.assertEqual(len(ids), 2)

    def test_gemini_semver_grouping(self):
        models = [
            ("gemini-2.0-flash", "google", "google", None),
            ("gemini-2.5-flash", "google", "google", None),
            ("gemini-3.0-flash", "google", "google", None),
        ]
        result = keep_latest_versions(models, 2)
        ids = [r[0] for r in result]
        self.assertIn("gemini-3.0-flash", ids)
        self.assertIn("gemini-2.5-flash", ids)
        self.assertNotIn("gemini-2.0-flash", ids)

    def test_empty_input(self):
        self.assertEqual(keep_latest_versions([], 2), [])


if __name__ == "__main__":
    unittest.main()
