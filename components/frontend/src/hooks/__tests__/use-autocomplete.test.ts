import { renderHook, act } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { useAutocomplete } from '../use-autocomplete';
import type { AutocompleteAgent, AutocompleteCommand } from '../use-autocomplete';
import React from 'react';

const mockAgents: AutocompleteAgent[] = [
  { id: 'a1', name: 'Agent Alpha', description: 'First agent' },
  { id: 'a2', name: 'Agent Beta', description: 'Second agent' },
  { id: 'a3', name: 'Agent Gamma' },
];

const mockCommands: AutocompleteCommand[] = [
  { id: 'c1', name: 'Help', slashCommand: '/help', description: 'Get help' },
  { id: 'c2', name: 'Clear', slashCommand: '/clear', description: 'Clear chat' },
  { id: 'c3', name: 'Status', slashCommand: '/status' },
];

describe('useAutocomplete', () => {
  const setup = () =>
    renderHook(() => useAutocomplete({ agents: mockAgents, commands: mockCommands }));

  describe('initial state', () => {
    it('starts closed with no items', () => {
      const { result } = setup();
      expect(result.current.isOpen).toBe(false);
      expect(result.current.type).toBeNull();
      expect(result.current.filteredItems).toEqual([]);
      expect(result.current.selectedIndex).toBe(0);
    });
  });

  describe('open and close', () => {
    it('opens for agents with @ trigger', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      expect(result.current.isOpen).toBe(true);
      expect(result.current.type).toBe('agent');
      expect(result.current.filteredItems).toHaveLength(3);
    });

    it('opens for commands with / trigger', () => {
      const { result } = setup();

      act(() => {
        result.current.open('command', 0);
      });

      expect(result.current.isOpen).toBe(true);
      expect(result.current.type).toBe('command');
      expect(result.current.filteredItems).toHaveLength(3);
    });

    it('closes and resets state', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      act(() => {
        result.current.close();
      });

      expect(result.current.isOpen).toBe(false);
      expect(result.current.type).toBeNull();
      expect(result.current.filter).toBe('');
      expect(result.current.selectedIndex).toBe(0);
    });
  });

  describe('filtering', () => {
    it('filters agents by name', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      act(() => {
        result.current.handleInputChange('@Alpha', 6);
      });

      expect(result.current.filteredItems).toHaveLength(1);
      expect(result.current.filteredItems[0].name).toBe('Agent Alpha');
    });

    it('filters agents by description', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      act(() => {
        result.current.handleInputChange('@Second', 7);
      });

      expect(result.current.filteredItems).toHaveLength(1);
      expect(result.current.filteredItems[0].name).toBe('Agent Beta');
    });

    it('filters commands by slash command', () => {
      const { result } = setup();

      act(() => {
        result.current.open('command', 0);
      });

      act(() => {
        result.current.handleInputChange('/hel', 4);
      });

      expect(result.current.filteredItems).toHaveLength(1);
      expect((result.current.filteredItems[0] as AutocompleteCommand).slashCommand).toBe('/help');
    });

    it('filters commands by description', () => {
      const { result } = setup();

      act(() => {
        result.current.open('command', 0);
      });

      act(() => {
        result.current.handleInputChange('/Clear', 6);
      });

      expect(result.current.filteredItems).toHaveLength(1);
    });

    it('resets selectedIndex on filter change', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
        result.current.setSelectedIndex(2);
      });

      act(() => {
        result.current.handleInputChange('@A', 2);
      });

      expect(result.current.selectedIndex).toBe(0);
    });
  });

  describe('handleInputChange triggers', () => {
    it('opens agent autocomplete on @ at start of input', () => {
      const { result } = setup();

      act(() => {
        result.current.handleInputChange('@', 1);
      });

      expect(result.current.isOpen).toBe(true);
      expect(result.current.type).toBe('agent');
    });

    it('opens command autocomplete on / at start of input', () => {
      const { result } = setup();

      act(() => {
        result.current.handleInputChange('/', 1);
      });

      expect(result.current.isOpen).toBe(true);
      expect(result.current.type).toBe('command');
    });

    it('opens after whitespace', () => {
      const { result } = setup();

      act(() => {
        result.current.handleInputChange('hello @', 7);
      });

      expect(result.current.isOpen).toBe(true);
      expect(result.current.type).toBe('agent');
    });

    it('does not open when @ is mid-word', () => {
      const { result } = setup();

      act(() => {
        result.current.handleInputChange('email@', 6);
      });

      expect(result.current.isOpen).toBe(false);
    });

    it('closes when cursor moves before trigger', () => {
      const { result } = setup();

      act(() => {
        result.current.handleInputChange('@', 1);
      });

      expect(result.current.isOpen).toBe(true);

      act(() => {
        result.current.handleInputChange('@', 0);
      });

      expect(result.current.isOpen).toBe(false);
    });

    it('closes when whitespace is typed in filter', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      act(() => {
        result.current.handleInputChange('@agent name', 11);
      });

      expect(result.current.isOpen).toBe(false);
    });
  });

  describe('handleKeyDown', () => {
    it('returns false when closed', () => {
      const { result } = setup();

      let consumed = false;
      act(() => {
        consumed = result.current.handleKeyDown({ key: 'ArrowDown', preventDefault: vi.fn() } as unknown as React.KeyboardEvent);
      });

      expect(consumed).toBe(false);
    });

    it('navigates down with ArrowDown', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      act(() => {
        result.current.handleKeyDown({ key: 'ArrowDown', preventDefault: vi.fn() } as unknown as React.KeyboardEvent);
      });

      expect(result.current.selectedIndex).toBe(1);
    });

    it('navigates up with ArrowUp', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
        result.current.setSelectedIndex(2);
      });

      act(() => {
        result.current.handleKeyDown({ key: 'ArrowUp', preventDefault: vi.fn() } as unknown as React.KeyboardEvent);
      });

      expect(result.current.selectedIndex).toBe(1);
    });

    it('does not go below 0 with ArrowUp', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      act(() => {
        result.current.handleKeyDown({ key: 'ArrowUp', preventDefault: vi.fn() } as unknown as React.KeyboardEvent);
      });

      expect(result.current.selectedIndex).toBe(0);
    });

    it('does not go past last item with ArrowDown', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
        result.current.setSelectedIndex(2);
      });

      act(() => {
        result.current.handleKeyDown({ key: 'ArrowDown', preventDefault: vi.fn() } as unknown as React.KeyboardEvent);
      });

      expect(result.current.selectedIndex).toBe(2);
    });

    it('consumes Tab key', () => {
      const { result } = setup();
      const preventDefault = vi.fn();

      act(() => {
        result.current.open('agent', 0);
      });

      let consumed = false;
      act(() => {
        consumed = result.current.handleKeyDown({ key: 'Tab', preventDefault } as unknown as React.KeyboardEvent);
      });

      expect(consumed).toBe(true);
      expect(preventDefault).toHaveBeenCalled();
    });

    it('consumes Enter key', () => {
      const { result } = setup();
      const preventDefault = vi.fn();

      act(() => {
        result.current.open('agent', 0);
      });

      let consumed = false;
      act(() => {
        consumed = result.current.handleKeyDown({ key: 'Enter', preventDefault } as unknown as React.KeyboardEvent);
      });

      expect(consumed).toBe(true);
      expect(preventDefault).toHaveBeenCalled();
    });

    it('closes on Escape', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      act(() => {
        result.current.handleKeyDown({ key: 'Escape', preventDefault: vi.fn() } as unknown as React.KeyboardEvent);
      });

      expect(result.current.isOpen).toBe(false);
    });

    it('returns false for unrelated keys', () => {
      const { result } = setup();

      act(() => {
        result.current.open('agent', 0);
      });

      let consumed = false;
      act(() => {
        consumed = result.current.handleKeyDown({ key: 'a', preventDefault: vi.fn() } as unknown as React.KeyboardEvent);
      });

      expect(consumed).toBe(false);
    });
  });

  describe('select', () => {
    it('inserts agent name and closes autocomplete', () => {
      const { result } = setup();
      const onChange = vi.fn();

      act(() => {
        result.current.open('agent', 0);
      });

      let newCursorPos = 0;
      act(() => {
        newCursorPos = result.current.select(mockAgents[0], '@Al', 3, onChange);
      });

      expect(onChange).toHaveBeenCalledWith('@Agent Alpha ');
      expect(newCursorPos).toBe(13);
      expect(result.current.isOpen).toBe(false);
    });

    it('inserts command slash text and closes autocomplete', () => {
      const { result } = setup();
      const onChange = vi.fn();

      act(() => {
        result.current.open('command', 0);
      });

      let newCursorPos = 0;
      act(() => {
        newCursorPos = result.current.select(mockCommands[0], '/hel', 4, onChange);
      });

      expect(onChange).toHaveBeenCalledWith('/help ');
      expect(newCursorPos).toBe(6);
      expect(result.current.isOpen).toBe(false);
    });

    it('preserves text after cursor', () => {
      const { result } = setup();
      const onChange = vi.fn();

      act(() => {
        result.current.open('agent', 5);
      });

      act(() => {
        result.current.select(mockAgents[0], 'hello @Al some text', 8, onChange);
      });

      // triggerPos=5, so textBefore = "hello", textAfter = "l some text" (from pos 8)
      expect(onChange).toHaveBeenCalledWith('hello@Agent Alpha l some text');
    });
  });
});
