'use client';

import * as React from 'react';
import { useCallback, useEffect, useId, useRef, useState } from 'react';
import { Loader2 } from 'lucide-react';

import { cn } from '@/lib/utils';
import { Input } from '@/components/ui/input';
import { useDebounce } from '@/hooks/use-debounce';
import { useLDAPUserSearch, useLDAPGroupSearch } from '@/services/queries';

type LDAPAutocompleteProps = {
  mode: 'user' | 'group';
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  id?: string;
  className?: string;
};

export function LDAPAutocomplete({
  mode,
  value,
  onChange,
  placeholder,
  disabled,
  id,
  className,
}: LDAPAutocompleteProps) {
  const instanceId = useId();
  const debouncedQuery = useDebounce(value.length >= 2 ? value : '', 300);
  const [isOpen, setIsOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const { data: users, isLoading: isLoadingUsers } = useLDAPUserSearch(
    mode === 'user' ? debouncedQuery : ''
  );
  const { data: groups, isLoading: isLoadingGroups } = useLDAPGroupSearch(
    mode === 'group' ? debouncedQuery : ''
  );

  const isLoading = mode === 'user' ? isLoadingUsers : isLoadingGroups;
  const hasResults = mode === 'user'
    ? (users && users.length > 0)
    : (groups && groups.length > 0);
  const resultCount = mode === 'user'
    ? (users?.length ?? 0)
    : (groups?.length ?? 0);

  // Show dropdown when we have a query and results
  useEffect(() => {
    if (debouncedQuery.length >= 2 && (hasResults || isLoading)) {
      setIsOpen(true);
    } else if (debouncedQuery.length < 2) {
      setIsOpen(false);
    }
  }, [debouncedQuery, hasResults, isLoading]);

  // Reset highlight when results change
  useEffect(() => {
    setHighlightedIndex(-1);
  }, [users, groups]);

  const selectItem = useCallback(
    (identifier: string) => {
      onChange(identifier);
      setIsOpen(false);
    },
    [onChange]
  );

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange(e.target.value);
    },
    [onChange]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!isOpen) return;

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setHighlightedIndex((prev) => (prev < resultCount - 1 ? prev + 1 : 0));
          break;
        case 'ArrowUp':
          e.preventDefault();
          setHighlightedIndex((prev) => (prev > 0 ? prev - 1 : resultCount - 1));
          break;
        case 'Enter':
          e.preventDefault();
          if (highlightedIndex >= 0) {
            if (mode === 'user' && users?.[highlightedIndex]) {
              selectItem(users[highlightedIndex].uid);
            } else if (mode === 'group' && groups?.[highlightedIndex]) {
              selectItem(groups[highlightedIndex].name);
            }
          }
          break;
        case 'Escape':
          setIsOpen(false);
          break;
      }
    },
    [isOpen, highlightedIndex, resultCount, mode, users, groups, selectItem]
  );

  // Close dropdown on click outside (only when open)
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  // Scroll highlighted item into view
  useEffect(() => {
    if (highlightedIndex >= 0 && listRef.current) {
      const items = listRef.current.querySelectorAll('[role="option"]');
      if (items[highlightedIndex]) {
        items[highlightedIndex].scrollIntoView({ block: 'nearest' });
      }
    }
  }, [highlightedIndex]);

  const listId = `${instanceId}-ldap-list`;

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <Input
          ref={inputRef}
          id={id}
          value={value}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={() => {
            if (debouncedQuery.length >= 2 && hasResults) {
              setIsOpen(true);
            }
          }}
          placeholder={placeholder}
          disabled={disabled}
          className={className}
          role="combobox"
          aria-expanded={isOpen}
          aria-autocomplete="list"
          aria-controls={isOpen ? listId : undefined}
          aria-activedescendant={
            highlightedIndex >= 0 ? `${instanceId}-option-${highlightedIndex}` : undefined
          }
          autoComplete="off"
        />
        {isLoading && debouncedQuery.length >= 2 && (
          <div className="absolute right-2 top-1/2 -translate-y-1/2">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        )}
      </div>

      {isOpen && (
        <ul
          ref={listRef}
          id={listId}
          role="listbox"
          className={cn(
            'absolute z-50 mt-1 w-full rounded-md border bg-popover text-popover-foreground shadow-md',
            'max-h-60 overflow-auto'
          )}
        >
          {isLoading && (
            <li className="px-3 py-2 text-sm text-muted-foreground flex items-center gap-2">
              <Loader2 className="h-3 w-3 animate-spin" />
              Searching...
            </li>
          )}

          {!isLoading && !hasResults && debouncedQuery.length >= 2 && (
            <li className="px-3 py-2 text-sm text-muted-foreground">No results found</li>
          )}

          {mode === 'user' &&
            users?.map((user, index) => (
              <li
                key={user.uid}
                id={`${instanceId}-option-${index}`}
                role="option"
                aria-selected={highlightedIndex === index}
                className={cn(
                  'cursor-pointer px-3 py-2 text-sm',
                  highlightedIndex === index
                    ? 'bg-accent text-accent-foreground'
                    : 'hover:bg-accent hover:text-accent-foreground'
                )}
                onMouseDown={(e) => {
                  e.preventDefault();
                  selectItem(user.uid);
                }}
                onMouseEnter={() => setHighlightedIndex(index)}
              >
                <div className="font-medium">
                  {user.fullName}
                  <span className="ml-1 text-muted-foreground">- {user.uid}</span>
                </div>
                {user.title && (
                  <div className="text-xs text-muted-foreground truncate">{user.title}</div>
                )}
              </li>
            ))}

          {mode === 'group' &&
            groups?.map((group, index) => (
              <li
                key={group.name}
                id={`${instanceId}-option-${index}`}
                role="option"
                aria-selected={highlightedIndex === index}
                className={cn(
                  'cursor-pointer px-3 py-2 text-sm',
                  highlightedIndex === index
                    ? 'bg-accent text-accent-foreground'
                    : 'hover:bg-accent hover:text-accent-foreground'
                )}
                onMouseDown={(e) => {
                  e.preventDefault();
                  selectItem(group.name);
                }}
                onMouseEnter={() => setHighlightedIndex(index)}
              >
                <div className="font-medium">{group.name}</div>
                {group.description && (
                  <div className="text-xs text-muted-foreground truncate">{group.description}</div>
                )}
              </li>
            ))}
        </ul>
      )}
    </div>
  );
}
