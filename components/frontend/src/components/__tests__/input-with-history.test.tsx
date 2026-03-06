import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { InputWithHistory } from '../input-with-history';

// Mock useInputHistory to control history data
vi.mock('@/hooks/use-input-history', () => ({
  useInputHistory: vi.fn().mockReturnValue({
    history: ['previous query', 'old search', 'another one'],
    addToHistory: vi.fn(),
  }),
}));

import { useInputHistory } from '@/hooks/use-input-history';

describe('InputWithHistory', () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let onChange: any;

  beforeEach(() => {
    onChange = vi.fn();
    vi.mocked(useInputHistory).mockReturnValue({
      history: ['previous query', 'old search', 'another one'],
      addToHistory: vi.fn(),
    });
  });

  it('renders an input element', () => {
    render(
      <InputWithHistory historyKey="test" value="" onChange={onChange} placeholder="Type here" />
    );
    expect(screen.getByPlaceholderText('Type here')).toBeDefined();
  });

  it('shows dropdown on focus when history items match', () => {
    render(
      <InputWithHistory historyKey="test" value="" onChange={onChange} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.focus(input);

    // All history items should appear since empty string matches everything
    expect(screen.getByText('previous query')).toBeDefined();
    expect(screen.getByText('old search')).toBeDefined();
  });

  it('filters history items based on current value', () => {
    render(
      <InputWithHistory historyKey="test" value="old" onChange={onChange} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.focus(input);

    expect(screen.getByText('old search')).toBeDefined();
    // "previous query" should not match "old"
    expect(screen.queryByText('previous query')).toBeNull();
  });

  it('selects item on click and calls onChange', () => {
    render(
      <InputWithHistory historyKey="test" value="" onChange={onChange} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.focus(input);

    const item = screen.getByText('previous query');
    fireEvent.mouseDown(item);

    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ target: { value: 'previous query' } })
    );
  });

  it('navigates dropdown with ArrowDown and ArrowUp', () => {
    render(
      <InputWithHistory historyKey="test" value="" onChange={onChange} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.focus(input);

    // ArrowDown to select first item
    fireEvent.keyDown(input, { key: 'ArrowDown' });
    // ArrowDown to select second item
    fireEvent.keyDown(input, { key: 'ArrowDown' });
    // ArrowUp to go back to first
    fireEvent.keyDown(input, { key: 'ArrowUp' });

    // Enter to select the highlighted item
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ target: { value: 'previous query' } })
    );
  });

  it('closes dropdown on Escape', () => {
    render(
      <InputWithHistory historyKey="test" value="" onChange={onChange} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.focus(input);

    expect(screen.getByText('previous query')).toBeDefined();

    fireEvent.keyDown(input, { key: 'Escape' });
    expect(screen.queryByText('previous query')).toBeNull();
  });

  it('calls parent onKeyDown for non-navigation keys', () => {
    const onKeyDown = vi.fn();
    render(
      <InputWithHistory historyKey="test" value="" onChange={onChange} onKeyDown={onKeyDown} />
    );
    const input = screen.getByRole('textbox');

    // Regular key should pass through
    fireEvent.keyDown(input, { key: 'a' });
    expect(onKeyDown).toHaveBeenCalled();
  });

  it('calls parent onFocus and onBlur', () => {
    const onFocus = vi.fn();
    const onBlur = vi.fn();
    render(
      <InputWithHistory
        historyKey="test"
        value=""
        onChange={onChange}
        onFocus={onFocus}
        onBlur={onBlur}
      />
    );
    const input = screen.getByRole('textbox');

    fireEvent.focus(input);
    expect(onFocus).toHaveBeenCalled();

    fireEvent.blur(input);
    expect(onBlur).toHaveBeenCalled();
  });

  it('does not show dropdown when no history items match', () => {
    render(
      <InputWithHistory historyKey="test" value="zzzzz no match" onChange={onChange} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.focus(input);

    // No items should match
    expect(screen.queryByText('previous query')).toBeNull();
    expect(screen.queryByText('old search')).toBeNull();
  });

  it('excludes current value from filtered history', () => {
    render(
      <InputWithHistory historyKey="test" value="previous query" onChange={onChange} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.focus(input);

    // "previous query" matches but equals currentValue, so it should be excluded
    expect(screen.queryByText('previous query')).toBeNull();
  });

  it('resets selected index on input change', () => {
    render(
      <InputWithHistory historyKey="test" value="" onChange={onChange} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.focus(input);

    // Navigate down
    fireEvent.keyDown(input, { key: 'ArrowDown' });

    // Type something — should reset index
    fireEvent.change(input, { target: { value: 'x' } });
    expect(onChange).toHaveBeenCalled();
  });
});
