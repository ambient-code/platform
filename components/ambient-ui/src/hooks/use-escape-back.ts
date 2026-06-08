'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

const INPUT_TAGS = new Set(['INPUT', 'TEXTAREA', 'SELECT'])

/**
 * Navigates back one level when the user presses Escape,
 * unless they are typing in an input or a modal/dialog is open.
 */
export function useEscapeBack() {
  const router = useRouter()

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Escape') return

      const target = e.target as Element | null
      if (target && INPUT_TAGS.has(target.tagName)) return

      // Don't fire when a dialog/sheet/overlay is open — those handle Escape themselves
      const openDialog = document.querySelector(
        '[data-state="open"][role="dialog"]',
      )
      if (openDialog) return

      // Also check for radix overlays
      const openOverlay = document.querySelector(
        '[data-slot="dialog-overlay"]',
      )
      if (openOverlay) return

      // Check for command palette (cmdk)
      const openCommand = document.querySelector('[cmdk-root]')
      if (openCommand) return

      e.preventDefault()
      router.back()
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [router])
}
