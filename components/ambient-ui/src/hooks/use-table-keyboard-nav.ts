'use client'

import { useCallback, useEffect, useState } from 'react'
import type { RefObject } from 'react'

const INPUT_TAGS = new Set(['INPUT', 'TEXTAREA', 'SELECT'])

type UseTableKeyboardNavOptions = {
  /** Total number of visible rows in the table */
  rowCount: number
  /** Called when the user presses Enter on the selected row */
  onSelect: (index: number) => void
  /** Ref to the table container element */
  containerRef: RefObject<HTMLElement | null>
}

/**
 * Manages keyboard navigation (j/k/ArrowUp/ArrowDown/Enter) for table rows.
 * Only activates when the container has focus or contains the active element.
 * Skips when an input element is focused.
 */
export function useTableKeyboardNav({
  rowCount,
  onSelect,
  containerRef,
}: UseTableKeyboardNavOptions) {
  const [selectedIndex, setSelectedIndex] = useState(-1)

  // Reset selection when row count changes (e.g. filter applied)
  useEffect(() => {
    setSelectedIndex((prev) => {
      if (rowCount === 0) return -1
      if (prev >= rowCount) return rowCount - 1
      return prev
    })
  }, [rowCount])

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const target = e.target as Element | null
      if (target && INPUT_TAGS.has(target.tagName)) return

      const container = containerRef.current
      if (!container) return

      // Only handle when focus is inside the container
      if (
        !container.contains(document.activeElement) &&
        document.activeElement !== container
      ) {
        return
      }

      if (rowCount === 0) return

      switch (e.key) {
        case 'j':
        case 'ArrowDown': {
          e.preventDefault()
          setSelectedIndex((prev) => {
            const next = prev < rowCount - 1 ? prev + 1 : prev
            return next
          })
          break
        }
        case 'k':
        case 'ArrowUp': {
          e.preventDefault()
          setSelectedIndex((prev) => {
            const next = prev > 0 ? prev - 1 : 0
            return next
          })
          break
        }
        case 'Enter': {
          if (selectedIndex >= 0 && selectedIndex < rowCount) {
            e.preventDefault()
            onSelect(selectedIndex)
          }
          break
        }
      }
    },
    [rowCount, selectedIndex, onSelect, containerRef],
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  return { selectedIndex, setSelectedIndex }
}
