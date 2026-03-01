#!/usr/bin/env python3
"""Ambient Platform SDK — Real-Time Session Watch Example"""

import os
import signal
import sys
import time
from datetime import datetime

from ambient_platform import AmbientClient


def main():
    print("Ambient Platform SDK — Real-Time Session Watch Example")
    print("=====================================================")
    print()

    try:
        # Create client
        client = AmbientClient.from_env()
        print("Connected to Ambient Platform")
        print(f"Project: {os.getenv('AMBIENT_PROJECT')}")
        print()

        # Set up signal handler for graceful shutdown
        interrupted = False
        def signal_handler(signum, frame):
            nonlocal interrupted
            print("\n\nReceived interrupt, stopping watch...")
            interrupted = True

        signal.signal(signal.SIGINT, signal_handler)

        # Start watching sessions
        print("Starting real-time watch for sessions...")
        print("Press Ctrl+C to stop.")
        print()

        with client.sessions.watch(timeout=1800.0) as watcher:
            for event in watcher.watch():
                if interrupted:
                    break
                
                handle_watch_event(event)

    except KeyboardInterrupt:
        print("\nWatch interrupted by user")
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    print("Watch ended")


def handle_watch_event(event):
    """Handle a session watch event."""
    timestamp = datetime.now().strftime("%H:%M:%S")
    
    if event.is_created():
        print(f"[{timestamp}] 🆕 CREATED session: {event.session.name} (id={event.resource_id})")
        if event.session.phase:
            print(f"        Phase: {event.session.phase}")
    elif event.is_updated():
        print(f"[{timestamp}] 📝 UPDATED session: {event.session.name} (id={event.resource_id})")
        if event.session.phase:
            print(f"        Phase: {event.session.phase}")
        if event.session.start_time:
            start_time = datetime.fromisoformat(event.session.start_time.replace('Z', '+00:00'))
            print(f"        Started: {start_time.strftime('%H:%M:%S')}")
    elif event.is_deleted():
        print(f"[{timestamp}] 🗑️  DELETED session: id={event.resource_id}")
    else:
        print(f"[{timestamp}] ❓ UNKNOWN event type: {event.type} (id={event.resource_id})")
    
    print()


async def async_watch_example():
    """Example using async watch functionality."""
    print("Async Session Watch Example")
    print("==========================")
    
    client = AmbientClient.from_env()
    
    async with client.sessions.watch_async(timeout=1800.0) as watcher:
        async for event in watcher.watch():
            handle_watch_event(event)


if __name__ == "__main__":
    main()