"""
Test cases for auth.map_to_vertex_model()

This module tests the model name mapping from Anthropic API model names
to Vertex AI model identifiers.
"""

import sys
from pathlib import Path

# Add parent directory to path for importing auth module
runner_dir = Path(__file__).parent.parent
if str(runner_dir) not in sys.path:
    sys.path.insert(0, str(runner_dir))

from auth import map_to_vertex_model  # type: ignore[import]


class TestMapToVertexModel:
    """Test suite for _map_to_vertex_model method"""

    def test_map_opus_4_6(self):
        """Test mapping for Claude Opus 4.6"""
        result = map_to_vertex_model("claude-opus-4-6")
        assert result == "claude-opus-4-6@default"

    def test_map_opus_4_5(self):
        """Test mapping for Claude Opus 4.5"""
        result = map_to_vertex_model("claude-opus-4-5")
        assert result == "claude-opus-4-5@20251101"

    def test_map_sonnet_4_5(self):
        """Test mapping for Claude Sonnet 4.5"""
        result = map_to_vertex_model("claude-sonnet-4-5")
        assert result == "claude-sonnet-4-5@20250929"

    def test_map_haiku_4_5(self):
        """Test mapping for Claude Haiku 4.5"""
        result = map_to_vertex_model("claude-haiku-4-5")
        assert result == "claude-haiku-4-5@20251001"

    def test_unknown_model_returns_unchanged(self):
        """Test that unknown model names are returned unchanged"""
        unknown_model = "claude-unknown-model-99"
        result = map_to_vertex_model(unknown_model)
        assert result == unknown_model

    def test_empty_string_returns_unchanged(self):
        """Test that empty string is returned unchanged"""
        result = map_to_vertex_model("")
        assert result == ""

    def test_case_sensitive_mapping(self):
        """Test that model mapping is case-sensitive"""

        # Uppercase should not match
        result = map_to_vertex_model("CLAUDE-OPUS-4-5")
        assert result == "CLAUDE-OPUS-4-5"  # Should return unchanged

    def test_whitespace_in_model_name(self):
        """Test handling of whitespace in model names"""

        # Model name with whitespace should not match
        result = map_to_vertex_model(" claude-opus-4-5 ")
        assert result == " claude-opus-4-5 "  # Should return unchanged

    def test_partial_model_name_no_match(self):
        """Test that partial model names don't match"""
        result = map_to_vertex_model("claude-opus")
        assert result == "claude-opus"  # Should return unchanged

    def test_vertex_model_id_passthrough(self):
        """Test that Vertex AI model IDs are returned unchanged"""
        vertex_id = "claude-opus-4-5@20251101"
        result = map_to_vertex_model(vertex_id)
        # If already a Vertex ID, should return unchanged
        assert result == vertex_id

    def test_all_frontend_models_have_mapping(self):
        """Test that all models from frontend dropdown have valid mappings"""

        # These are the exact model values from the frontend dropdown
        frontend_models = [
            "claude-sonnet-4-5",
            "claude-opus-4-6",
            "claude-opus-4-5",
            "claude-haiku-4-5",
        ]

        expected_mappings = {
            "claude-sonnet-4-5": "claude-sonnet-4-5@20250929",
            "claude-opus-4-6": "claude-opus-4-6@default",
            "claude-opus-4-5": "claude-opus-4-5@20251101",
            "claude-haiku-4-5": "claude-haiku-4-5@20251001",
        }

        for model in frontend_models:
            result = map_to_vertex_model(model)
            assert result == expected_mappings[model], (
                f"Model {model} should map to {expected_mappings[model]}, got {result}"
            )

    def test_mapping_includes_version_suffix(self):
        """Test that all mapped models include version suffixes"""

        models_with_date = [
            "claude-opus-4-5",
            "claude-sonnet-4-5",
            "claude-haiku-4-5",
        ]

        for model in models_with_date:
            result = map_to_vertex_model(model)
            assert "@" in result, (
                f"Mapped model {result} should include @ version suffix"
            )
            assert len(result.split("@")) == 2, (
                f"Mapped model {result} should have exactly one @"
            )
            version_date = result.split("@")[1]
            assert len(version_date) == 8, (
                f"Version date {version_date} should be 8 digits (YYYYMMDD)"
            )
            assert version_date.isdigit(), (
                f"Version date {version_date} should be all digits"
            )

        # Opus 4.6 uses @default instead of a date suffix
        result = map_to_vertex_model("claude-opus-4-6")
        assert result == "claude-opus-4-6@default"

    def test_none_input_handling(self):
        """Test that None input passes through unchanged (dict.get handles it)"""
        result = map_to_vertex_model(None)  # type: ignore[arg-type]
        assert result is None

    def test_numeric_input_handling(self):
        """Test that numeric input passes through unchanged (dict.get handles it)"""
        result = map_to_vertex_model(123)  # type: ignore[arg-type]
        assert result == 123

    def test_mapping_consistency(self):
        """Test that mapping is consistent across multiple calls"""

        model = "claude-sonnet-4-5"

        # Call multiple times
        results = [map_to_vertex_model(model) for _ in range(5)]

        # All results should be identical
        assert all(r == results[0] for r in results)
        assert results[0] == "claude-sonnet-4-5@20250929"


class TestModelMappingIntegration:
    """Integration tests for model mapping in realistic scenarios"""

    def test_mapping_matches_available_vertex_models(self):
        """Test that mapped model IDs match the expected Vertex AI format"""

        # Expected Vertex AI model ID format: model-name@YYYYMMDD or model-name@default
        models_to_test = [
            ("claude-opus-4-6", "claude-opus-4-6@default"),
            ("claude-opus-4-5", "claude-opus-4-5@20251101"),
            ("claude-sonnet-4-5", "claude-sonnet-4-5@20250929"),
            ("claude-haiku-4-5", "claude-haiku-4-5@20251001"),
        ]

        for input_model, expected_vertex_id in models_to_test:
            result = map_to_vertex_model(input_model)
            assert result == expected_vertex_id, (
                f"Expected {input_model} to map to {expected_vertex_id}, got {result}"
            )

    def test_ui_to_vertex_round_trip(self):
        """Test that UI model selection properly maps to Vertex AI"""

        # Simulate user selecting from UI dropdown
        ui_selections = [
            "claude-sonnet-4-5",  # User selects Sonnet 4.5
            "claude-opus-4-6",  # User selects Opus 4.6
            "claude-opus-4-5",  # User selects Opus 4.5
            "claude-haiku-4-5",  # User selects Haiku 4.5
        ]

        for selection in ui_selections:
            vertex_model = map_to_vertex_model(selection)

            # Verify it maps to a valid Vertex AI model ID
            assert vertex_model.startswith("claude-")
            assert "@" in vertex_model

            # Verify the base model name is preserved
            base_name = vertex_model.split("@")[0]
            assert base_name == selection

    def test_end_to_end_vertex_mapping_flow(self):
        """Test complete flow: UI selection → model mapping → Vertex AI call"""

        # Simulate complete flow for each model
        test_scenarios = [
            {
                "ui_selection": "claude-opus-4-6",
                "expected_vertex_id": "claude-opus-4-6@default",
                "description": "Latest Opus model",
            },
            {
                "ui_selection": "claude-opus-4-5",
                "expected_vertex_id": "claude-opus-4-5@20251101",
                "description": "Previous Opus model",
            },
            {
                "ui_selection": "claude-sonnet-4-5",
                "expected_vertex_id": "claude-sonnet-4-5@20250929",
                "description": "Balanced model",
            },
            {
                "ui_selection": "claude-haiku-4-5",
                "expected_vertex_id": "claude-haiku-4-5@20251001",
                "description": "Fastest model",
            },
        ]

        for scenario in test_scenarios:
            # Step 1: User selects model from UI
            ui_model = scenario["ui_selection"]

            # Step 2: Backend maps to Vertex AI model ID
            vertex_model_id = map_to_vertex_model(ui_model)

            # Step 3: Verify correct mapping
            assert vertex_model_id == scenario["expected_vertex_id"], (
                f"{scenario['description']}: Expected {scenario['expected_vertex_id']}, got {vertex_model_id}"
            )

            # Step 4: Verify Vertex AI model ID format is valid
            assert "@" in vertex_model_id
            parts = vertex_model_id.split("@")
            assert len(parts) == 2
            model_name, version_suffix = parts
            assert model_name.startswith("claude-")
            # Version suffix is either "default" or YYYYMMDD date
            assert version_suffix == "default" or (
                len(version_suffix) == 8 and version_suffix.isdigit()
            ), f"Version suffix {version_suffix} should be 'default' or 8-digit date"

    def test_model_ordering_consistency(self):
        """Test that model ordering is consistent between frontend and backend"""

        # Expected ordering: Sonnet → Opus 4.6 → Opus 4.5 → Haiku (matches frontend dropdown)
        expected_order = [
            "claude-sonnet-4-5",
            "claude-opus-4-6",
            "claude-opus-4-5",
            "claude-haiku-4-5",
        ]

        # Verify all models map successfully in order
        for model in expected_order:
            vertex_id = map_to_vertex_model(model)
            assert "@" in vertex_id, f"Model {model} should map to valid Vertex AI ID"

        # Verify ordering matches frontend dropdown
        assert expected_order[0] == "claude-sonnet-4-5"  # Balanced (default)
        assert expected_order[1] == "claude-opus-4-6"  # Latest Opus
        assert expected_order[2] == "claude-opus-4-5"  # Previous Opus
        assert expected_order[3] == "claude-haiku-4-5"  # Fastest
