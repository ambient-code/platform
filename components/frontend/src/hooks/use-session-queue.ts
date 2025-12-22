"use client";

import { useState, useEffect, useCallback } from "react";

// Data structures
export interface QueuedMessageItem {
  id: string;              // Unique ID for deduplication
  content: string;         // Message text
  timestamp: number;       // When queued
  sentAt?: number;         // When sent (if sent)
}

export interface QueuedWorkflowItem {
  id: string;              // Workflow ID
  gitUrl: string;
  branch: string;
  path: string;
  timestamp: number;
  activatedAt?: number;    // When activated (if activated)
}

export interface QueueMetadata {
  sessionPhase: string;    // Last known phase
  lastPolled: number;      // Last poll timestamp
  processing: boolean;     // Currently processing queue
}

const STORAGE_KEY_PREFIX = "vteam:queue";
const MAX_AGE_MS = 24 * 60 * 60 * 1000; // 24 hours
const MAX_MESSAGES = 100;

/**
 * Hook to manage session-specific queues in localStorage
 * Handles messages and workflows that need to be sent when session becomes Running
 */
export function useSessionQueue(projectName: string, sessionName: string) {
  const [messages, setMessages] = useState<QueuedMessageItem[]>([]);
  const [workflow, setWorkflow] = useState<QueuedWorkflowItem | null>(null);
  const [metadata, setMetadata] = useState<QueueMetadata>({
    sessionPhase: "Unknown",
    lastPolled: Date.now(),
    processing: false,
  });

  // Generate storage keys
  const getStorageKeys = useCallback(() => {
    const base = `${STORAGE_KEY_PREFIX}:${projectName}:${sessionName}`;
    return {
      messages: `${base}:messages`,
      workflow: `${base}:workflow`,
      metadata: `${base}:meta`,
    };
  }, [projectName, sessionName]);

  // Check if localStorage is available
  const isLocalStorageAvailable = useCallback(() => {
    try {
      const test = '__localStorage_test__';
      localStorage.setItem(test, test);
      localStorage.removeItem(test);
      return true;
    } catch {
      return false;
    }
  }, []);

  // Load data from localStorage
  const loadFromStorage = useCallback(() => {
    if (!isLocalStorageAvailable()) {
      console.warn('localStorage not available, queue will not persist');
      return;
    }

    const keys = getStorageKeys();
    const now = Date.now();

    try {
      // Load messages
      const messagesRaw = localStorage.getItem(keys.messages);
      if (messagesRaw) {
        const parsed = JSON.parse(messagesRaw) as QueuedMessageItem[];
        // Filter out old messages (older than 24h)
        const fresh = parsed.filter(m => now - m.timestamp < MAX_AGE_MS);
        // Limit to max count
        const limited = fresh.slice(-MAX_MESSAGES);
        setMessages(limited);
        
        // Clean up if we filtered anything
        if (limited.length !== parsed.length) {
          localStorage.setItem(keys.messages, JSON.stringify(limited));
        }
      }

      // Load workflow
      const workflowRaw = localStorage.getItem(keys.workflow);
      if (workflowRaw) {
        const parsed = JSON.parse(workflowRaw) as QueuedWorkflowItem;
        // Check if workflow is fresh
        if (now - parsed.timestamp < MAX_AGE_MS) {
          setWorkflow(parsed);
        } else {
          localStorage.removeItem(keys.workflow);
        }
      }

      // Load metadata
      const metadataRaw = localStorage.getItem(keys.metadata);
      if (metadataRaw) {
        const parsed = JSON.parse(metadataRaw) as QueueMetadata;
        setMetadata(parsed);
      }
    } catch (error) {
      console.error('Failed to load queue from localStorage:', error);
      // Clear corrupted data
      try {
        localStorage.removeItem(keys.messages);
        localStorage.removeItem(keys.workflow);
        localStorage.removeItem(keys.metadata);
      } catch {
        // Ignore cleanup errors
      }
    }
  }, [getStorageKeys, isLocalStorageAvailable]);

  // Save data to localStorage
  const saveToStorage = useCallback((
    newMessages?: QueuedMessageItem[],
    newWorkflow?: QueuedWorkflowItem | null,
    newMetadata?: QueueMetadata
  ) => {
    if (!isLocalStorageAvailable()) {
      return;
    }

    const keys = getStorageKeys();

    try {
      if (newMessages !== undefined) {
        localStorage.setItem(keys.messages, JSON.stringify(newMessages));
      }
      if (newWorkflow !== undefined) {
        if (newWorkflow === null) {
          localStorage.removeItem(keys.workflow);
        } else {
          localStorage.setItem(keys.workflow, JSON.stringify(newWorkflow));
        }
      }
      if (newMetadata !== undefined) {
        localStorage.setItem(keys.metadata, JSON.stringify(newMetadata));
      }
    } catch (error) {
      console.error('Failed to save queue to localStorage:', error);
      // Handle quota exceeded
      if (error instanceof DOMException && error.name === 'QuotaExceededError') {
        console.warn('localStorage quota exceeded, clearing old data');
        try {
          // Clear messages to free up space
          localStorage.removeItem(keys.messages);
          setMessages([]);
        } catch {
          // Ignore
        }
      }
    }
  }, [getStorageKeys, isLocalStorageAvailable]);

  // Load on mount
  useEffect(() => {
    loadFromStorage();
  }, [loadFromStorage]);

  // Add message to queue
  const addMessage = useCallback((content: string) => {
    const newMessage: QueuedMessageItem = {
      id: `msg-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      content,
      timestamp: Date.now(),
    };

    setMessages(prev => {
      const updated = [...prev, newMessage];
      saveToStorage(updated);
      return updated;
    });
  }, [saveToStorage]);

  // Get all messages
  const getMessages = useCallback(() => {
    return messages;
  }, [messages]);

  // Mark message as sent
  const markMessageSent = useCallback((id: string) => {
    setMessages(prev => {
      const updated = prev.map(m => 
        m.id === id ? { ...m, sentAt: Date.now() } : m
      );
      saveToStorage(updated);
      return updated;
    });
  }, [saveToStorage]);

  // Clear all messages
  const clearMessages = useCallback(() => {
    setMessages([]);
    saveToStorage([]);
  }, [saveToStorage]);

  // Set workflow
  const setQueuedWorkflow = useCallback((workflowConfig: {
    id: string;
    gitUrl: string;
    branch: string;
    path: string;
  }) => {
    const newWorkflow: QueuedWorkflowItem = {
      ...workflowConfig,
      timestamp: Date.now(),
    };
    setWorkflow(newWorkflow);
    saveToStorage(undefined, newWorkflow);
  }, [saveToStorage]);

  // Get workflow
  const getWorkflow = useCallback(() => {
    return workflow;
  }, [workflow]);

  // Mark workflow as activated
  const markWorkflowActivated = useCallback((id: string) => {
    if (workflow && workflow.id === id) {
      const updated = { ...workflow, activatedAt: Date.now() };
      setWorkflow(updated);
      saveToStorage(undefined, updated);
    }
  }, [workflow, saveToStorage]);

  // Clear workflow
  const clearWorkflow = useCallback(() => {
    setWorkflow(null);
    saveToStorage(undefined, null);
  }, [saveToStorage]);

  // Get metadata
  const getMetadata = useCallback(() => {
    return metadata;
  }, [metadata]);

  // Update metadata
  const updateMetadata = useCallback((updates: Partial<QueueMetadata>) => {
    setMetadata(prev => {
      const updated = { ...prev, ...updates };
      saveToStorage(undefined, undefined, updated);
      return updated;
    });
  }, [saveToStorage]);

  return {
    // Message operations
    addMessage,
    getMessages,
    markMessageSent,
    clearMessages,
    
    // Workflow operations
    setWorkflow: setQueuedWorkflow,
    getWorkflow,
    markWorkflowActivated,
    clearWorkflow,
    
    // Metadata operations
    getMetadata,
    updateMetadata,
    
    // Direct state access (for React rendering)
    messages,
    workflow,
    metadata,
  };
}

