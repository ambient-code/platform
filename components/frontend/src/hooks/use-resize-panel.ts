import { useCallback, useEffect, useRef, useState } from "react";
import { useLocalStorage } from "./use-local-storage";

/**
 * Hook for drag-to-resize panel behavior.
 * Returns a width value, a setter, a mousedown handler, and an isDragging flag
 * so the parent can disable CSS transitions during drag.
 */
export function useResizePanel(
  storageKey: string,
  defaultWidth: number,
  minWidth: number,
  maxWidth: number,
  side: "left" | "right" = "left"
) {
  const [width, setWidth] = useLocalStorage(storageKey, defaultWidth);
  const [isDragging, setIsDragging] = useState(false);
  const dragging = useRef(false);
  const listenersRef = useRef<{ move: (ev: MouseEvent) => void; up: () => void } | null>(null);

  // Cleanup listeners on unmount to prevent memory leaks
  useEffect(() => {
    return () => {
      if (listenersRef.current) {
        document.removeEventListener("mousemove", listenersRef.current.move);
        document.removeEventListener("mouseup", listenersRef.current.up);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        listenersRef.current = null;
      }
    };
  }, []);

  const onMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      dragging.current = true;
      setIsDragging(true);
      const startX = e.clientX;
      const startWidth = width;

      const onMouseMove = (ev: MouseEvent) => {
        if (!dragging.current) return;
        const delta = side === "left" ? ev.clientX - startX : startX - ev.clientX;
        const newWidth = Math.min(maxWidth, Math.max(minWidth, startWidth + delta));
        setWidth(newWidth);
      };

      const onMouseUp = () => {
        dragging.current = false;
        setIsDragging(false);
        document.removeEventListener("mousemove", onMouseMove);
        document.removeEventListener("mouseup", onMouseUp);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        listenersRef.current = null;
      };

      // Store refs for cleanup
      listenersRef.current = { move: onMouseMove, up: onMouseUp };

      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
      document.addEventListener("mousemove", onMouseMove);
      document.addEventListener("mouseup", onMouseUp);
    },
    [width, minWidth, maxWidth, side, setWidth]
  );

  return { width, setWidth, onMouseDown, isDragging };
}
