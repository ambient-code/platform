// Ambient Platform SDK — Real-Time Session Watch Example

import { AmbientClient } from '../src/client';
import { SessionWatchEventUtils } from '../src/session_watch';

async function main() {
  console.log('Ambient Platform SDK — Real-Time Session Watch Example');
  console.log('=====================================================');
  console.log();

  try {
    // Create client from environment variables
    const client = AmbientClient.fromEnv();
    
    console.log('Connected to Ambient Platform');
    console.log(`Project: ${process.env.AMBIENT_PROJECT}`);
    console.log();

    // Set up signal handler for graceful shutdown
    let interrupted = false;
    const handleSignal = () => {
      console.log('\n\nReceived interrupt, stopping watch...');
      interrupted = true;
    };
    
    process.on('SIGINT', handleSignal);
    process.on('SIGTERM', handleSignal);

    // Start watching sessions
    console.log('Starting real-time watch for sessions...');
    console.log('Press Ctrl+C to stop.');
    console.log();

    const watcher = client.sessions.watch({
      timeout: 30 * 60 * 1000, // 30 minutes
    });

    try {
      for await (const event of watcher.watch()) {
        if (interrupted) {
          break;
        }
        
        handleWatchEvent(event);
      }
    } finally {
      watcher.close();
    }

  } catch (error) {
    console.error('Error:', error);
    process.exit(1);
  }

  console.log('Watch ended');
}

function handleWatchEvent(event: any) {
  const timestamp = new Date().toLocaleTimeString();
  
  if (SessionWatchEventUtils.isCreated(event)) {
    console.log(`[${timestamp}] 🆕 CREATED session: ${event.session?.name} (id=${event.resourceId})`);
    if (event.session?.phase) {
      console.log(`        Phase: ${event.session.phase}`);
    }
  } else if (SessionWatchEventUtils.isUpdated(event)) {
    console.log(`[${timestamp}] 📝 UPDATED session: ${event.session?.name} (id=${event.resourceId})`);
    if (event.session?.phase) {
      console.log(`        Phase: ${event.session.phase}`);
    }
    if (event.session?.startTime) {
      console.log(`        Started: ${event.session.startTime.toLocaleTimeString()}`);
    }
  } else if (SessionWatchEventUtils.isDeleted(event)) {
    console.log(`[${timestamp}] 🗑️  DELETED session: id=${event.resourceId}`);
  } else {
    console.log(`[${timestamp}] ❓ UNKNOWN event type: ${event.type} (id=${event.resourceId})`);
  }
  
  console.log();
}

// Browser-compatible example
export async function browserWatchExample() {
  console.log('Browser Session Watch Example');
  console.log('=============================');
  
  const client = new AmbientClient({
    baseUrl: window.location.origin,
    token: localStorage.getItem('ambient_token') || '',
    project: localStorage.getItem('ambient_project') || '',
  });
  
  const controller = new AbortController();
  const watcher = client.sessions.watch({
    signal: controller.signal,
    timeout: 30 * 60 * 1000,
  });

  // Stop watching after 5 minutes for demo
  setTimeout(() => {
    controller.abort();
  }, 5 * 60 * 1000);

  try {
    for await (const event of watcher.watch()) {
      console.log('Session event:', event);
    }
  } catch (error) {
    if (error.name !== 'AbortError') {
      console.error('Watch error:', error);
    }
  } finally {
    watcher.close();
  }
}

if (require.main === module) {
  main().catch(console.error);
}