import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useResizeTextarea } from '../use-resize-textarea';

describe('useResizeTextarea', () => {
  it('returns default height', () => {
    const { result } = renderHook(() => useResizeTextarea());
    expect(result.current.textareaHeight).toBe(108);
  });

  it('accepts custom default height', () => {
    const { result } = renderHook(() =>
      useResizeTextarea({ defaultHeight: 200 }),
    );
    expect(result.current.textareaHeight).toBe(200);
  });

  it('provides handleResizeStart function', () => {
    const { result } = renderHook(() => useResizeTextarea());
    expect(typeof result.current.handleResizeStart).toBe('function');
  });

  it('clamps height to maxHeight when dragging up', () => {
    const { result } = renderHook(() =>
      useResizeTextarea({ defaultHeight: 100, minHeight: 50, maxHeight: 150 }),
    );

    // Simulate mouse drag: start at y=500, drag up to y=100 (delta=400)
    const mouseDownEvent = {
      preventDefault: () => {},
      clientY: 500,
    } as React.MouseEvent;

    act(() => {
      result.current.handleResizeStart(mouseDownEvent);
    });

    // Simulate mousemove via document event
    act(() => {
      const moveEvent = new MouseEvent('mousemove', { clientY: 100 });
      document.dispatchEvent(moveEvent);
    });

    // Should be clamped to maxHeight (150)
    expect(result.current.textareaHeight).toBe(150);

    // Clean up
    act(() => {
      document.dispatchEvent(new MouseEvent('mouseup'));
    });
  });

  it('clamps height to minHeight when dragging down', () => {
    const { result } = renderHook(() =>
      useResizeTextarea({ defaultHeight: 100, minHeight: 50, maxHeight: 300 }),
    );

    const mouseDownEvent = {
      preventDefault: () => {},
      clientY: 200,
    } as React.MouseEvent;

    act(() => {
      result.current.handleResizeStart(mouseDownEvent);
    });

    // Drag down: y increases, delta becomes negative
    act(() => {
      const moveEvent = new MouseEvent('mousemove', { clientY: 500 });
      document.dispatchEvent(moveEvent);
    });

    // Should be clamped to minHeight (50)
    expect(result.current.textareaHeight).toBe(50);

    act(() => {
      document.dispatchEvent(new MouseEvent('mouseup'));
    });
  });
});
