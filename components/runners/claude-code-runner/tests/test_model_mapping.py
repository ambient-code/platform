"""
Test cases for ClaudeCodeAdapter._map_to_vertex_model()

This module tests the model name mapping from Anthropic API model names
to Vertex AI model identifiers.
"""

import sys
from pathlib import Path

import pytest

# Add parent directory to path for importing adapter module
adapter_dir = Path(__file__).parent.parent
if str(adapter_dir) not in sys.path:
    sys.path.insert(0, str(adapter_dir))

from adapter import ClaudeCodeAdapter  # type: ignore[import]


class TestMapToVertexModel:
    """Test suite for _map_to_vertex_model method"""

    def test_map_opus_4_5(self):
        """Test mapping for Claude Opus 4.5"""
        adapter = ClaudeCodeAdapter()
        result = adapter._map_to_vertex_model("claude-opus-4-5")
        assert result == "claude-opus-4-5@20251101"

    def test_map_opus_4_1(self):
        """Test mapping for Claude Opus 4.1"""
        adapter = ClaudeCodeAdapter()
        result = adapter._map_to_vertex_model("claude-opus-4-1")
        assert result == "claude-opus-4-1@20250805"

    def test_map_sonnet_4_5(self):
        """Test mapping for Claude Sonnet 4.5"""
        adapter = ClaudeCodeAdapter()
        result = adapter._map_to_vertex_model("claude-sonnet-4-5")
        assert result == "claude-sonnet-4-5@20250929"

    def test_map_haiku_4_5(self):
        """Test mapping for Claude Haiku 4.5"""
        adapter = ClaudeCodeAdapter()
        result = adapter._map_to_vertex_model("claude-haiku-4-5")
        assert result == "claude-haiku-4-5@20251001"

    def test_unknown_model_returns_unchanged(self):
        """Test that unknown model names are returned unchanged"""
        adapter = ClaudeCodeAdapter()
        unknown_model = "claude-unknown-model-99"
        result = adapter._map_to_vertex_model(unknown_model)
        assert result == unknown_model

    def test_empty_string_returns_unchanged(self):
        """Test that empty string is returned unchanged"""
        adapter = ClaudeCodeAdapter()
        result = adapter._map_to_vertex_model("")
        assert result == ""

    def test_case_sensitive_mapping(self):
        """Test that model mapping is case-sensitive"""
        adapter = ClaudeCodeAdapter()
        # Uppercase should not match
        result = adapter._map_to_vertex_model("CLAUDE-OPUS-4-1")
        assert result == "CLAUDE-OPUS-4-1"  # Should return unchanged

    def test_whitespace_in_model_name(self):
        """Test handling of whitespace in model names"""
        adapter = ClaudeCodeAdapter()
        # Model name with whitespace should not match
        result = adapter._map_to_vertex_model(" claude-opus-4-1 ")
        assert result == " claude-opus-4-1 "  # Should return unchanged

    def test_partial_model_name_no_match(self):
        """Test that partial model names don't match"""
        adapter = ClaudeCodeAdapter()
        result = adapter._map_to_vertex_model("claude-opus")
        assert result == "claude-opus"  # Should return unchanged

    def test_vertex_model_id_passthrough(self):
        """Test that Vertex AI model IDs are returned unchanged"""
        adapter = ClaudeCodeAdapter()
        vertex_id = "claude-opus-4-1@20250805"
        result = adapter._map_to_vertex_model(vertex_id)
        # If already a Vertex ID, should return unchanged
        assert result == vertex_id

    def test_all_frontend_models_have_mapping(self):
        """Test that all models from frontend dropdown have valid mappings"""
        adapter = ClaudeCodeAdapter()

        # These are the exact model values from the frontend dropdown
        frontend_models = [
            "claude-sonnet-4-5",
            "claude-opus-4-6",
            "claude-opus-4-5",
            "claude-opus-4-1",
            "claude-haiku-4-5",
        ]

        expected_mappings = {
            "claude-sonnet-4-5": "claude-sonnet-4-5@20250929",
            "claude-opus-4-6": "claude-opus-4-6",
            "claude-opus-4-5": "claude-opus-4-5@20251101",
            "claude-opus-4-1": "claude-opus-4-1@20250805",
            "claude-haiku-4-5": "claude-haiku-4-5@20251001",
        }

        for model in frontend_models:
            result = adapter._map_to_vertex_model(model)
            assert (
                result == expected_mappings[model]
            ), f"Model {model} should map to {expected_mappings[model]}, got {result}"

    def test_mapping_includes_version_date(self):
        """Test that mapped models include version dates (except Opus 4.6)"""
        adapter = ClaudeCodeAdapter()

        # Opus 4.6 is the exception - uses simplified naming without @date
        models_with_dates = [
            "claude-opus-4-5",
            "claude-opus-4-1",
            "claude-sonnet-4-5",
            "claude-haiku-4-5",
        ]

        for model in models_with_dates:
            result = adapter._map_to_vertex_model(model)
            # All Vertex AI models (except Opus 4.6) should have @YYYYMMDD format
            assert "@" in result, f"Mapped model {result} should include @ version date"
            assert (
                len(result.split("@")) == 2
            ), f"Mapped model {result} should have exactly one @"
            version_date = result.split("@")[1]
            assert (
                len(version_date) == 8
            ), f"Version date {version_date} should be 8 digits (YYYYMMDD)"
            assert (
                version_date.isdigit()
            ), f"Version date {version_date} should be all digits"

    def test_opus_4_6_no_date_suffix(self):
        """Test that Opus 4.6 uses simplified naming (no @date suffix)"""
        adapter = ClaudeCodeAdapter()
        result = adapter._map_to_vertex_model("claude-opus-4-6")
        assert result == "claude-opus-4-6"
        assert "@" not in result, "Opus 4.6 should NOT have @date suffix"

    def test_none_input_handling(self):
        """Test that None input raises TypeError (invalid type per signature)"""
        adapter = ClaudeCodeAdapter()
        # Function signature specifies str -> str, so None should raise
        with pytest.raises((TypeError, AttributeError)):
            adapter._map_to_vertex_model(None)  # type: ignore[arg-type]

    def test_numeric_input_handling(self):
        """Test that numeric input raises TypeError (invalid type per signature)"""
        adapter = ClaudeCodeAdapter()
        # Function signature specifies str -> str, so int should raise
        with pytest.raises((TypeError, AttributeError)):
            adapter._map_to_vertex_model(123)  # type: ignore[arg-type]

    def test_mapping_consistency(self):
        """Test that mapping is consistent across multiple calls"""
        adapter = ClaudeCodeAdapter()
        model = "claude-sonnet-4-5"

        # Call multiple times
        results = [adapter._map_to_vertex_model(model) for _ in range(5)]

        # All results should be identical
        assert all(r == results[0] for r in results)
        assert results[0] == "claude-sonnet-4-5@20250929"


class TestModelMappingIntegration:
    """Integration tests for model mapping in realistic scenarios"""

    def test_mapping_matches_available_vertex_models(self):
        """Test that mapped model IDs match the expected Vertex AI format"""
        adapter = ClaudeCodeAdapter()

        # Expected Vertex AI model ID format: model-name@YYYYMMDD (except Opus 4.6)
        models_to_test = [
            ("claude-opus-4-6", "claude-opus-4-6"),
            ("claude-opus-4-5", "claude-opus-4-5@20251101"),
            ("claude-opus-4-1", "claude-opus-4-1@20250805"),
            ("claude-sonnet-4-5", "claude-sonnet-4-5@20250929"),
            ("claude-haiku-4-5", "claude-haiku-4-5@20251001"),
        ]

        for input_model, expected_vertex_id in models_to_test:
            result = adapter._map_to_vertex_model(input_model)
            assert (
                result == expected_vertex_id
            ), f"Expected {input_model} to map to {expected_vertex_id}, got {result}"

    def test_ui_to_vertex_round_trip(self):
        """Test that UI model selection properly maps to Vertex AI"""
        adapter = ClaudeCodeAdapter()

        # Simulate user selecting from UI dropdown
        ui_selections = [
            "claude-sonnet-4-5",  # User selects Sonnet 4.5
            "claude-opus-4-6",  # User selects Opus 4.6 (newest)
            "claude-opus-4-5",  # User selects Opus 4.5
            "claude-opus-4-1",  # User selects Opus 4.1
            "claude-haiku-4-5",  # User selects Haiku 4.5
        ]

        for selection in ui_selections:
            vertex_model = adapter._map_to_vertex_model(selection)

            # Verify it maps to a valid Vertex AI model ID
            assert vertex_model.startswith("claude-")

            # Opus 4.6 is the exception - no @date suffix
            if selection == "claude-opus-4-6":
                assert "@" not in vertex_model
            else:
                assert "@" in vertex_model

            # Verify the base model name is preserved
            if "@" in vertex_model:
                base_name = vertex_model.split("@")[0]
                assert selection in vertex_model or base_name in selection
            else:
                assert selection == vertex_model

    def test_end_to_end_vertex_mapping_flow(self):
        """Test complete flow: UI selection → model mapping → Vertex AI call"""
        adapter = ClaudeCodeAdapter()

        # Simulate complete flow for each model
        test_scenarios = [
            {
                "ui_selection": "claude-opus-4-6",
                "expected_vertex_id": "claude-opus-4-6",
                "description": "Newest Opus model (simplified naming)",
                "has_date_suffix": False,
            },
            {
                "ui_selection": "claude-opus-4-5",
                "expected_vertex_id": "claude-opus-4-5@20251101",
                "description": "Latest Opus model",
                "has_date_suffix": True,
            },
            {
                "ui_selection": "claude-opus-4-1",
                "expected_vertex_id": "claude-opus-4-1@20250805",
                "description": "Previous Opus model",
                "has_date_suffix": True,
            },
            {
                "ui_selection": "claude-sonnet-4-5",
                "expected_vertex_id": "claude-sonnet-4-5@20250929",
                "description": "Balanced model",
                "has_date_suffix": True,
            },
            {
                "ui_selection": "claude-haiku-4-5",
                "expected_vertex_id": "claude-haiku-4-5@20251001",
                "description": "Fastest model",
                "has_date_suffix": True,
            },
        ]

        for scenario in test_scenarios:
            # Step 1: User selects model from UI
            ui_model = scenario["ui_selection"]

            # Step 2: Backend maps to Vertex AI model ID
            vertex_model_id = adapter._map_to_vertex_model(ui_model)

            # Step 3: Verify correct mapping
            assert (
                vertex_model_id == scenario["expected_vertex_id"]
            ), f"{scenario['description']}: Expected {scenario['expected_vertex_id']}, got {vertex_model_id}"

            # Step 4: Verify Vertex AI model ID format is valid
            if scenario["has_date_suffix"]:
                assert "@" in vertex_model_id
                parts = vertex_model_id.split("@")
                assert len(parts) == 2
                model_name, version_date = parts
                assert model_name.startswith("claude-")
                assert len(version_date) == 8  # YYYYMMDD format
                assert version_date.isdigit()
            else:
                # Opus 4.6 uses simplified naming - no @date suffix
                assert "@" not in vertex_model_id
                assert vertex_model_id.startswith("claude-")

    def test_model_ordering_consistency(self):
        """Test that model ordering is consistent between frontend and backend"""
        adapter = ClaudeCodeAdapter()

        # Expected ordering: Sonnet → Opus 4.6 → Opus 4.5 → Opus 4.1 → Haiku (matches frontend dropdown)
        expected_order = [
            "claude-sonnet-4-5",
            "claude-opus-4-6",
            "claude-opus-4-5",
            "claude-opus-4-1",
            "claude-haiku-4-5",
        ]

        # Verify all models map successfully in order
        for model in expected_order:
            vertex_id = adapter._map_to_vertex_model(model)
            # Opus 4.6 is the exception - no @date suffix
            if model == "claude-opus-4-6":
                assert (
                    "@" not in vertex_id
                ), f"Model {model} should use simplified naming"
            else:
                assert (
                    "@" in vertex_id
                ), f"Model {model} should map to valid Vertex AI ID"

        # Verify ordering matches frontend dropdown
        assert expected_order[0] == "claude-sonnet-4-5"  # Balanced (default)
        assert expected_order[1] == "claude-opus-4-6"  # Newest Opus
        assert expected_order[2] == "claude-opus-4-5"  # Latest Opus
        assert expected_order[3] == "claude-opus-4-1"  # Previous Opus
        assert expected_order[4] == "claude-haiku-4-5"  # Fastest
