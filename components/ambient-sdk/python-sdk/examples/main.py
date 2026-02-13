#!/usr/bin/env python3
"""
Ambient Platform SDK - HTTP Client Example

This example demonstrates how to use the Ambient Platform Python SDK to interact 
with the platform via HTTP API.
"""

import os
import sys
from typing import Optional

from ambient_platform import (
    AmbientClient,
    CreateSessionRequest,
    RepoHTTP,
    StatusCompleted,
    StatusFailed,
)
from ambient_platform.exceptions import (
    AmbientAPIError,
    AmbientConnectionError,
    SessionNotFoundError,
    AuthenticationError,
)

# Example session configuration
EXAMPLE_TASK = "Analyze the repository structure and provide a brief summary of the codebase organization."
EXAMPLE_MODEL = "claude-3.5-sonnet"
DEFAULT_API_URL = "http://localhost:8080"


def get_env_or_default(key: str, default: str = "") -> str:
    """Get environment variable value or default if not set."""
    return os.getenv(key, default)


def should_monitor_session() -> bool:
    """Check if user wants to monitor session completion."""
    monitor = get_env_or_default("MONITOR_SESSION", "false").lower()
    return monitor in ("true", "1", "yes")


def truncate_string(s: str, max_len: int) -> str:
    """Truncate a string to specified length with ellipsis."""
    if len(s) <= max_len:
        return s
    return s[:max_len-3] + "..."


def print_session_details(session):
    """Print detailed information about a session."""
    print(f"   ID: {session.id}")
    print(f"   Status: {session.status}")
    print(f"   Task: {truncate_string(session.task, 80)}")
    print(f"   Model: {session.model}")
    print(f"   Created: {session.created_at}")
    
    if session.completed_at:
        print(f"   Completed: {session.completed_at}")
    
    if session.result:
        print(f"   Result: {truncate_string(session.result, 100)}")
    
    if session.error:
        print(f"   Error: {session.error}")


def main():
    """Main example function."""
    print("🐍 Ambient Platform SDK - Python HTTP Client Example")
    print("===================================================")

    # Get configuration from environment or use defaults
    api_url = get_env_or_default("AMBIENT_API_URL", DEFAULT_API_URL)
    token = get_env_or_default("AMBIENT_TOKEN")
    project = get_env_or_default("AMBIENT_PROJECT", "")

    if not token:
        print("❌ AMBIENT_TOKEN environment variable is required")
        sys.exit(1)

    try:
        # Create HTTP client
        with AmbientClient(api_url, token, project, timeout=60.0) as client:
            print(f"✓ Created client for API: {api_url}")
            print(f"✓ Using project: {project}")

            # Example 1: Create a new session
            print("\n📝 Creating new session...")
            create_req = CreateSessionRequest(
                task=EXAMPLE_TASK,
                model=EXAMPLE_MODEL,
                repos=[
                    RepoHTTP(
                        url="https://github.com/ambient-code/platform",
                        branch="main",
                    )
                ],
            )

            create_resp = client.create_session(create_req)
            session_id = create_resp.id
            print(f"✓ Created session: {session_id}")

            # Example 2: Get session details
            print("\n🔍 Getting session details...")
            session = client.get_session(session_id)
            print_session_details(session)

            # Example 3: List all sessions
            print("\n📋 Listing all sessions...")
            list_resp = client.list_sessions()
            print(f"✓ Found {len(list_resp.items)} sessions (total: {list_resp.total})")
            
            for i, s in enumerate(list_resp.items[:3]):  # Show first 3 sessions
                print(f"  {i+1}. {s.id} ({s.status}) - {truncate_string(s.task, 60)}")
            
            if len(list_resp.items) > 3:
                print(f"  ... and {len(list_resp.items) - 3} more")

            # Example 4: Monitor session (optional)
            if should_monitor_session():
                print("\n⏳ Monitoring session completion...")
                print("   Note: This may take time depending on the task complexity")
                
                try:
                    completed_session = client.wait_for_completion(
                        session_id, poll_interval=5.0, timeout=300.0  # 5 minutes max
                    )
                    print("\n🎉 Session completed!")
                    print_session_details(completed_session)
                except TimeoutError:
                    print("⏰ Monitoring timed out after 5 minutes")
                except Exception as e:
                    print(f"❌ Monitoring failed: {e}")

            print("\n✅ Python HTTP Client demonstration complete!")
            print("\n💡 Next steps:")
            print("   • Check session status periodically")
            print("   • Use the session ID to retrieve results")
            print("   • Create additional sessions as needed")

    except AuthenticationError as e:
        print(f"❌ Authentication failed: {e}")
        print("   Check your AMBIENT_TOKEN environment variable")
        sys.exit(1)
    except AmbientConnectionError as e:
        print(f"❌ Connection failed: {e}")
        print("   Check your AMBIENT_API_URL and network connectivity")
        sys.exit(1)
    except SessionNotFoundError as e:
        print(f"❌ Session not found: {e}")
        sys.exit(1)
    except AmbientAPIError as e:
        print(f"❌ API error: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()