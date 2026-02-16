"use client";

import { useState, useCallback, useRef } from "react";

type UseResizeTextareaOptions = {
  defaultHeight?: number;
  minHeight?: number;
  maxHeight?: number;
};

export function useResizeTextarea(options: UseResizeTextareaOptions = {}) {
  const { defaultHeight = 108, minHeight = 60, maxHeight = 300 } = options;
  const [textareaHeight, setTextareaHeight] = useState(defaultHeight);
  const startYRef = useRef(0);
  const startHeightRef = useRef(0);

  const handleResizeStart = useCallback(
    (e: React.MouseEvent | React.TouchEvent) => {
      e.preventDefault();
      const clientY = "touches" in e ? e.touches[0].clientY : e.clientY;
      startYRef.current = clientY;
      startHeightRef.current = textareaHeight;

      const onMove = (ev: MouseEvent | TouchEvent) => {
        const currentY =
          "touches" in ev
            ? (ev as TouchEvent).touches[0].clientY
            : (ev as MouseEvent).clientY;
        // Dragging up increases height, dragging down decreases
        const delta = startYRef.current - currentY;
        const newHeight = Math.min(
          maxHeight,
          Math.max(minHeight, startHeightRef.current + delta)
        );
        setTextareaHeight(newHeight);
      };

      const onEnd = () => {
        document.removeEventListener("mousemove", onMove);
        document.removeEventListener("mouseup", onEnd);
        document.removeEventListener("touchmove", onMove);
        document.removeEventListener("touchend", onEnd);
      };

      document.addEventListener("mousemove", onMove);
      document.addEventListener("mouseup", onEnd);
      document.addEventListener("touchmove", onMove);
      document.addEventListener("touchend", onEnd);
    },
    [textareaHeight, minHeight, maxHeight]
  );

  return { textareaHeight, handleResizeStart };
}
