#!/usr/bin/env python3
"""
Test script to verify model metadata is being tracked in Langfuse.

This test validates that:
1. Model is included in session-level metadata via propagate_attributes
2. Model is properly tracked in individual generations
3. Model information is visible in Langfuse UI
"""

import asyncio
import os
import sys
import logging
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from observability import ObservabilityManager

logging.basicConfig(level=logging.INFO)


async def test_model_metadata_tracking():
    """Test that model metadata is properly tracked in Langfuse."""

    # Test configuration
    session_id = "test-session-model-metadata"
    user_id = "test-user"
    user_name = "Test User"
    namespace = "test-namespace"
    prompt = "Test prompt for model metadata tracking"

    # Test different model configurations
    test_models = [
        "claude-sonnet-4-5@20250929",
        "claude-haiku-4-5@20251001",
        "claude-opus-4-1@20250805",
        None  # Test with no model specified
    ]

    for test_model in test_models:
        print(f"\n{'='*60}")
        print(f"Testing with model: {test_model}")
        print('='*60)

        # Create observability manager
        obs = ObservabilityManager(
            session_id=f"{session_id}-{test_model or 'default'}",
            user_id=user_id,
            user_name=user_name
        )

        # Initialize with model metadata
        success = await obs.initialize(
            prompt=prompt,
            namespace=namespace,
            model=test_model
        )

        if success:
            print(f"✓ Observability initialized successfully with model: {test_model}")

            # Simulate tracking an interaction
            test_message = type('Message', (), {
                'content': [type('TextBlock', (), {'text': 'Test response'})]
            })()

            test_usage = {
                'input_tokens': 100,
                'output_tokens': 50,
                'cache_read_input_tokens': 20,
                'cache_creation_input_tokens': 10
            }

            obs.track_interaction(
                message=test_message,
                model=test_model or 'claude-sonnet-4-5@20250929',
                turn_count=1,
                usage=test_usage
            )

            print(f"✓ Tracked interaction with model: {test_model}")

            # Finalize
            await obs.finalize()
            print(f"✓ Finalized observability session")

        else:
            print(f"✗ Failed to initialize observability (expected if Langfuse not configured)")

    print("\n" + "="*60)
    print("Model metadata tracking test completed!")
    print("Check Langfuse UI to verify:")
    print("1. Sessions are grouped by session_id")
    print("2. Model appears in session metadata")
    print("3. Model is shown for each generation")
    print("="*60)


async def test_propagate_attributes_with_model():
    """Test that propagate_attributes includes model in metadata."""

    # This test requires Langfuse to be configured
    if not all([
        os.getenv("LANGFUSE_ENABLED") == "true",
        os.getenv("LANGFUSE_PUBLIC_KEY"),
        os.getenv("LANGFUSE_SECRET_KEY"),
        os.getenv("LANGFUSE_HOST")
    ]):
        print("Skipping propagate_attributes test - Langfuse not configured")
        print("To run this test, set:")
        print("  LANGFUSE_ENABLED=true")
        print("  LANGFUSE_PUBLIC_KEY=<your-key>")
        print("  LANGFUSE_SECRET_KEY=<your-secret>")
        print("  LANGFUSE_HOST=<langfuse-url>")
        return

    from langfuse import Langfuse, propagate_attributes

    # Initialize client
    client = Langfuse(
        public_key=os.getenv("LANGFUSE_PUBLIC_KEY"),
        secret_key=os.getenv("LANGFUSE_SECRET_KEY"),
        host=os.getenv("LANGFUSE_HOST")
    )

    # Test propagate_attributes with model in metadata
    test_model = "claude-sonnet-4-5@20250929"

    with client.start_as_current_observation(as_type="trace", name="test-model-metadata"):
        with propagate_attributes(
            user_id="test-user",
            session_id="test-session-propagate",
            metadata={
                "model": test_model,
                "namespace": "test",
                "test_type": "model_metadata"
            },
            tags=["model-metadata-test", f"model:{test_model}"]
        ):
            # Create a generation that should inherit the metadata
            with client.start_as_current_observation(
                as_type="generation",
                name="test-generation",
                model=test_model
            ) as gen:
                gen.update(
                    input="Test input",
                    output="Test output with model metadata",
                    usage_details={
                        "input": 10,
                        "output": 20
                    }
                )

    # Flush data
    client.flush()
    print(f"✓ Created trace with model metadata: {test_model}")
    print("Check Langfuse UI to verify model appears in trace metadata")


if __name__ == "__main__":
    print("Testing Langfuse Model Metadata Tracking")
    print("="*60)

    # Run tests
    asyncio.run(test_model_metadata_tracking())
    asyncio.run(test_propagate_attributes_with_model())