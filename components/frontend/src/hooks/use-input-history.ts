import { useCallback } from 'react';
import { useLocalStorage } from './use-local-storage';

const MAX_HISTORY_ITEMS = 10;

export function useInputHistory(storageKey: string, maxItems = MAX_HISTORY_ITEMS) {
  const [history, setHistory] = useLocalStorage<string[]>(
    `form-input-history:${storageKey}`,
    []
  );

  const addToHistory = useCallback(
    (value: string) => {
      const trimmed = value.trim();
      if (!trimmed) return;
      setHistory((prev) => {
        const filtered = prev.filter((item) => item !== trimmed);
        return [trimmed, ...filtered].slice(0, maxItems);
      });
    },
    [setHistory, maxItems]
  );

  return { history, addToHistory };
}
