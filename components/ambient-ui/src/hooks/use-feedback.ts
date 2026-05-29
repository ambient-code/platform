'use client'

import { useCallback, useState } from 'react'
import type { FeedbackItem } from '@/domain/types'

export type UseFeedbackReturn = {
  feedbackMode: boolean
  pendingItems: FeedbackItem[]
  sentItems: FeedbackItem[]
  enterFeedbackMode: () => void
  exitFeedbackMode: () => void
  addItem: (item: FeedbackItem) => void
  removeItem: (id: string) => void
  updateComment: (id: string, comment: string) => void
  markAsSent: () => void
  clearPending: () => void
}

export function useFeedback(): UseFeedbackReturn {
  const [feedbackMode, setFeedbackMode] = useState(false)
  const [pendingItems, setPendingItems] = useState<FeedbackItem[]>([])
  const [sentItems, setSentItems] = useState<FeedbackItem[]>([])

  const enterFeedbackMode = useCallback(() => {
    setFeedbackMode(true)
  }, [])

  const exitFeedbackMode = useCallback(() => {
    setFeedbackMode(false)
  }, [])

  const addItem = useCallback((item: FeedbackItem) => {
    setPendingItems((prev) => [...prev, item])
  }, [])

  const removeItem = useCallback((id: string) => {
    setPendingItems((prev) => prev.filter((item) => item.id !== id))
  }, [])

  const updateComment = useCallback((id: string, comment: string) => {
    setPendingItems((prev) =>
      prev.map((item) => (item.id === id ? { ...item, comment } : item))
    )
  }, [])

  const markAsSent = useCallback(() => {
    setSentItems((sent) => [...sent, ...pendingItems])
    setPendingItems([])
  }, [pendingItems])

  const clearPending = useCallback(() => {
    setPendingItems([])
  }, [])

  return {
    feedbackMode,
    pendingItems,
    sentItems,
    enterFeedbackMode,
    exitFeedbackMode,
    addItem,
    removeItem,
    updateComment,
    markAsSent,
    clearPending,
  }
}
