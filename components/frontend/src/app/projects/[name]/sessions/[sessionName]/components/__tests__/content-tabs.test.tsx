import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ContentTabs } from '../content-tabs';
import type { FileTab, ActiveTab } from '../../hooks/use-file-tabs';

describe('ContentTabs', () => {
  const defaultProps = {
    openTabs: [] as FileTab[],
    activeTab: { type: 'chat' } as ActiveTab,
    onSwitchToChat: vi.fn(),
    onSwitchToFile: vi.fn(),
    onCloseFile: vi.fn(),
  };

  it('renders Chat tab always', () => {
    render(<ContentTabs {...defaultProps} />);
    expect(screen.getByText('Chat')).toBeDefined();
  });

  it('renders file tabs when openTabs provided', () => {
    const tabs: FileTab[] = [
      { path: '/src/index.ts', name: 'index.ts' },
      { path: '/src/app.tsx', name: 'app.tsx' },
    ];
    render(<ContentTabs {...defaultProps} openTabs={tabs} />);

    expect(screen.getByText('index.ts')).toBeDefined();
    expect(screen.getByText('app.tsx')).toBeDefined();
  });

  it('calls onSwitchToChat when Chat tab clicked', () => {
    const onSwitchToChat = vi.fn();
    render(<ContentTabs {...defaultProps} onSwitchToChat={onSwitchToChat} />);

    fireEvent.click(screen.getByText('Chat'));
    expect(onSwitchToChat).toHaveBeenCalledTimes(1);
  });

  it('calls onSwitchToFile when file tab clicked', () => {
    const onSwitchToFile = vi.fn();
    const tabs: FileTab[] = [{ path: '/src/index.ts', name: 'index.ts' }];
    render(
      <ContentTabs
        {...defaultProps}
        openTabs={tabs}
        onSwitchToFile={onSwitchToFile}
      />
    );

    fireEvent.click(screen.getByText('index.ts'));
    expect(onSwitchToFile).toHaveBeenCalledTimes(1);
    expect(onSwitchToFile).toHaveBeenCalledWith('/src/index.ts');
  });

  it('calls onCloseFile when close button clicked', () => {
    const onCloseFile = vi.fn();
    const tabs: FileTab[] = [{ path: '/src/index.ts', name: 'index.ts' }];
    render(
      <ContentTabs
        {...defaultProps}
        openTabs={tabs}
        onCloseFile={onCloseFile}
      />
    );

    const closeButton = screen.getByRole('button', { name: 'Close index.ts' });
    fireEvent.click(closeButton);
    expect(onCloseFile).toHaveBeenCalledTimes(1);
    expect(onCloseFile).toHaveBeenCalledWith('/src/index.ts');
  });

  it('renders rightActions when provided', () => {
    render(
      <ContentTabs
        {...defaultProps}
        rightActions={<button type="button">Settings</button>}
      />
    );

    expect(screen.getByText('Settings')).toBeDefined();
  });

  it('shows no file tabs when openTabs is empty', () => {
    const { container } = render(<ContentTabs {...defaultProps} openTabs={[]} />);

    // Only the Chat button should exist as a tab button
    const buttons = container.querySelectorAll('button');
    // Should only have the Chat button (no close buttons or file tab buttons)
    expect(buttons.length).toBe(1);
    expect(screen.getByText('Chat')).toBeDefined();
  });
});
