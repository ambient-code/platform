/**
 * useSessionQueue hook
 * 
 * Manages a localStorage-backed queue for messages and workflows that need to be
 * processed when a session transitions from Pending/Creating to Running state.
 * 
 * This allows users to:
 * 1. Queue messages while a session is starting up
 * 2. Queue workflow selections before session is ready
 * 3. Automatically process queued items when session becomes Running
 */

import { useState, useCallback, useEffect } from 'react';

// Types
export interface QueuedMessageItem {
  id: string;
  content: string;
  timestamp: number;
  sentAt?: number;
}

export interface QueuedWorkflowItem {
  id: string;
  gitUrl: string;
  branch: string;
  path: string;
  timestamp: number;
  activatedAt?: number;
}

export interface QueueMetadata {
  sessionPhase?: string;
  processing?: boolean;
  lastProcessedAt?: number;
  retryCount?: number;
  errorCount?: number;
  lastError?: string;
}

interface UseSessionQueueReturn {
  // Message queue operations
  messages: QueuedMessageItem[];
  addMessage: (content: string) => void;
  markMessageSent: (messageId: string) => void;
  clearMessages: () => void;
  
  // Workflow queue operations
  workflow: QueuedWorkflowItem | null;
  setWorkflow: (workflow: Omit<QueuedWorkflowItem, 'timestamp'>) => void;
  markWorkflowActivated: (workflowId: string) => void;
  clearWorkflow: () => void;
  
  // Metadata operations
  metadata: QueueMetadata;
  updateMetadata: (updates: Partial<QueueMetadata>) => void;
}

// Constants
const MAX_MESSAGES = 100;
const MESSAGE_RETENTION_MS = 24 * 60 * 60 * 1000; // 24 hours

// Generate unique ID
function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

// Filter out old messages
function filterOldMessages(messages: QueuedMessageItem[]): QueuedMessageItem[] {
  const now = Date.now();
  return messages.filter(msg => (now - msg.timestamp) < MESSAGE_RETENTION_MS);
}

/**
 * Hook to manage session queue (messages and workflows)
 */
export function useSessionQueue(
  projectName: string,
  sessionName: string
): UseSessionQueueReturn {
  const messagesKey = `vteam:queue:${projectName}:${sessionName}:messages`;
  const workflowKey = `vteam:queue:${projectName}:${sessionName}:workflow`;
  const metadataKey = `vteam:queue:${projectName}:${sessionName}:metadata`;

  // Initialize state from localStorage
  const [messages, setMessages] = useState<QueuedMessageItem[]>(() => {
    if (typeof window === 'undefined') return [];
    try {
      const stored = localStorage.getItem(messagesKey);
      if (!stored) return [];
      const parsed = JSON.parse(stored) as QueuedMessageItem[];
      return filterOldMessages(parsed).slice(-MAX_MESSAGES);
    } catch (error) {
      console.warn('Failed to load queued messages from localStorage:', error);
      return [];
    }
  });

  const [workflow, setWorkflowState] = useState<QueuedWorkflowItem | null>(() => {
    if (typeof window === 'undefined') return null;
    try {
      const stored = localStorage.getItem(workflowKey);
      if (!stored) return null;
      return JSON.parse(stored) as QueuedWorkflowItem;
    } catch (error) {
      console.warn('Failed to load queued workflow from localStorage:', error);
      return null;
    }
  });

  const [metadata, setMetadata] = useState<QueueMetadata>(() => {
    if (typeof window === 'undefined') return {};
    try {
      const stored = localStorage.getItem(metadataKey);
      if (!stored) return {};
      return JSON.parse(stored) as QueueMetadata;
    } catch (error) {
      console.warn('Failed to load queue metadata from localStorage:', error);
      return {};
    }
  });

  // Persist messages to localStorage
  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      if (messages.length === 0) {
        localStorage.removeItem(messagesKey);
      } else {
        localStorage.setItem(messagesKey, JSON.stringify(messages));
      }
    } catch (error) {
      console.warn('Failed to persist messages to localStorage:', error);
    }
  }, [messages, messagesKey]);

  // Persist workflow to localStorage
  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      if (workflow === null) {
        localStorage.removeItem(workflowKey);
      } else {
        localStorage.setItem(workflowKey, JSON.stringify(workflow));
      }
    } catch (error) {
      console.warn('Failed to persist workflow to localStorage:', error);
    }
  }, [workflow, workflowKey]);

  // Persist metadata to localStorage
  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      // Clean up if metadata is empty
      if (Object.keys(metadata).length === 0) {
        localStorage.removeItem(metadataKey);
      } else {
        localStorage.setItem(metadataKey, JSON.stringify(metadata));
      }
    } catch (error) {
      console.warn('Failed to persist metadata to localStorage:', error);
    }
  }, [metadata, metadataKey]);

  // Message operations
  const addMessage = useCallback((content: string) => {
    const newMessage: QueuedMessageItem = {
      id: generateId(),
      content,
      timestamp: Date.now(),
    };
    setMessages(prev => [...filterOldMessages(prev), newMessage].slice(-MAX_MESSAGES));
  }, []);

  const markMessageSent = useCallback((messageId: string) => {
    setMessages(prev => 
      prev.map(msg => 
        msg.id === messageId 
          ? { ...msg, sentAt: Date.now() }
          : msg
      )
    );
  }, []);

  const clearMessages = useCallback(() => {
    setMessages([]);
  }, []);

  // Workflow operations
  const setWorkflow = useCallback((workflowData: Omit<QueuedWorkflowItem, 'timestamp'>) => {
    setWorkflowState({
      ...workflowData,
      timestamp: Date.now(),
    });
  }, []);

  const markWorkflowActivated = useCallback((workflowId: string) => {
    setWorkflowState(prev => 
      prev && prev.id === workflowId
        ? { ...prev, activatedAt: Date.now() }
        : prev
    );
  }, []);

  const clearWorkflow = useCallback(() => {
    setWorkflowState(null);
  }, []);

  // Metadata operations
  const updateMetadata = useCallback((updates: Partial<QueueMetadata>) => {
    setMetadata(prev => ({ ...prev, ...updates }));
  }, []);

  return {
    messages,
    addMessage,
    markMessageSent,
    clearMessages,
    workflow,
    setWorkflow,
    markWorkflowActivated,
    clearWorkflow,
    metadata,
    updateMetadata,
  };
}

