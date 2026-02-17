/**
 * useDraftInput hook
 *
 * Manages localStorage-backed persistence for chat input drafts.
 *
 * This allows users to:
 * 1. Preserve typed messages when navigating away from a session
 * 2. Restore draft content when returning to the session
 * 3. Automatically clear drafts when messages are sent
 */

import { useState, useCallback, useEffect } from 'react';

type UseDraftInputReturn = {
  draft: string;
  setDraft: (value: string) => void;
  clearDraft: () => void;
};

/**
 * Hook to manage session draft input with localStorage persistence
 */
export function useDraftInput(
  projectName: string,
  sessionName: string
): UseDraftInputReturn {
  const draftKey = `vteam:draft:${projectName}:${sessionName}:input`;

  // Initialize state from localStorage
  const [draft, setDraftState] = useState<string>(() => {
    if (typeof window === 'undefined') return '';
    try {
      const stored = localStorage.getItem(draftKey);
      return stored || '';
    } catch (error) {
      console.warn('Failed to load draft input from localStorage:', error);
      return '';
    }
  });

  // Persist draft to localStorage
  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      if (draft === '') {
        localStorage.removeItem(draftKey);
      } else {
        localStorage.setItem(draftKey, draft);
      }
    } catch (error) {
      console.warn('Failed to persist draft input to localStorage:', error);
    }
  }, [draft, draftKey]);

  // Update draft value
  const setDraft = useCallback((value: string) => {
    setDraftState(value);
  }, []);

  // Clear draft
  const clearDraft = useCallback(() => {
    setDraftState('');
  }, []);

  return {
    draft,
    setDraft,
    clearDraft,
  };
}
