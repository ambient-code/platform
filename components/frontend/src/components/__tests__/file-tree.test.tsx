/* eslint-disable @typescript-eslint/no-unused-vars */
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { FileTree, type FileTreeNode } from '../file-tree';

function makeTree(): FileTreeNode[] {
  return [
    {
      name: 'src',
      path: '/src',
      type: 'folder',
      children: [
        { name: 'index.ts', path: '/src/index.ts', type: 'file', sizeKb: 1.5 },
        {
          name: 'components',
          path: '/src/components',
          type: 'folder',
          children: [
            { name: 'Button.tsx', path: '/src/components/Button.tsx', type: 'file' },
          ],
        },
      ],
    },
    { name: 'README.md', path: '/README.md', type: 'file', sizeKb: 2.3 },
  ];
}

describe('FileTree', () => {
  it('renders top-level nodes', () => {
    const onSelect = vi.fn();
    const { container } = render(<FileTree nodes={makeTree()} onSelect={onSelect} />);

    expect(screen.getByText('src')).toBeDefined();
    expect(screen.getByText('README.md')).toBeDefined();
  });

  it('renders nested children when folder is expanded (default)', () => {
    const onSelect = vi.fn();
    const { container } = render(<FileTree nodes={makeTree()} onSelect={onSelect} />);

    // Children should be visible since expanded defaults to true
    expect(screen.getByText('index.ts')).toBeDefined();
    expect(screen.getByText('components')).toBeDefined();
    expect(screen.getByText('Button.tsx')).toBeDefined();
  });

  it('displays file size when sizeKb is set', () => {
    const onSelect = vi.fn();
    const { container } = render(<FileTree nodes={makeTree()} onSelect={onSelect} />);

    expect(screen.getByText('1.5K')).toBeDefined();
    expect(screen.getByText('2.3K')).toBeDefined();
  });

  it('calls onSelect when a file is clicked', () => {
    const onSelect = vi.fn();
    const { container } = render(<FileTree nodes={makeTree()} onSelect={onSelect} />);

    fireEvent.click(screen.getByText('README.md'));
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenCalledWith(
      expect.objectContaining({ path: '/README.md', type: 'file' })
    );
  });

  it('collapses and re-expands folder on click', () => {
    const onSelect = vi.fn();
    const { container } = render(<FileTree nodes={makeTree()} onSelect={onSelect} />);

    // Children visible initially
    expect(screen.getByText('index.ts')).toBeDefined();

    // Click folder to collapse
    fireEvent.click(screen.getByText('src'));
    expect(screen.queryByText('index.ts')).toBeNull();

    // Click again to expand
    fireEvent.click(screen.getByText('src'));
    expect(screen.getByText('index.ts')).toBeDefined();
  });

  it('calls onToggle when expanding a folder', () => {
    const onSelect = vi.fn();
    const onToggle = vi.fn();
    const { container } = render(<FileTree nodes={makeTree()} onSelect={onSelect} onToggle={onToggle} />);

    // First collapse
    fireEvent.click(screen.getByText('src'));
    expect(onToggle).not.toHaveBeenCalled();

    // Then expand again — onToggle should fire
    fireEvent.click(screen.getByText('src'));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it('highlights selected path', () => {
    const onSelect = vi.fn();
    const { container } = render(
      <FileTree nodes={makeTree()} selectedPath="/README.md" onSelect={onSelect} />
    );

    // The selected item should have the bg-muted class and font-medium
    const selectedItem = screen.getByText('README.md');
    expect(selectedItem.className).toContain('font-medium');
  });

  it('calls onSelect for empty folder (no children)', () => {
    const onSelect = vi.fn();
    const nodes: FileTreeNode[] = [
      { name: 'empty-dir', path: '/empty', type: 'folder', children: [] },
    ];
    const { container } = render(<FileTree nodes={nodes} onSelect={onSelect} />);

    fireEvent.click(screen.getByText('empty-dir'));
    expect(onSelect).toHaveBeenCalledWith(
      expect.objectContaining({ path: '/empty', type: 'folder' })
    );
  });

  it('displays branch badge when branch is set', () => {
    const onSelect = vi.fn();
    const nodes: FileTreeNode[] = [
      { name: 'repo', path: '/repo', type: 'folder', branch: 'main', children: [] },
    ];
    const { container } = render(<FileTree nodes={nodes} onSelect={onSelect} />);

    expect(screen.getByText('main')).toBeDefined();
  });

  it('renders with className prop', () => {
    const onSelect = vi.fn();
    const { container } = render(
      <FileTree nodes={[]} onSelect={onSelect} className="custom-class" />
    );

    expect((container.firstChild as HTMLElement).className).toContain('custom-class');
  });
});
