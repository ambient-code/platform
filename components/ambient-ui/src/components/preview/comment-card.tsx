'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent } from '@/components/ui/card'

export type CommentCardProps = {
  type: 'element' | 'region'
  position: { x: number; y: number }
  dimensions?: { width: number; height: number }
  capturedHtml?: string
  onSubmit: (comment: string) => void
  onCancel: () => void
}

export function CommentCard({
  type,
  position,
  dimensions,
  capturedHtml,
  onSubmit,
  onCancel,
}: CommentCardProps) {
  const [comment, setComment] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    // Focus textarea when card mounts
    const timer = setTimeout(() => textareaRef.current?.focus(), 50)
    return () => clearTimeout(timer)
  }, [])

  const handleSubmit = useCallback(() => {
    if (comment.trim()) {
      onSubmit(comment.trim())
    }
  }, [comment, onSubmit])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        handleSubmit()
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        onCancel()
      }
    },
    [handleSubmit, onCancel]
  )

  const selectionLabel =
    type === 'region' && dimensions
      ? `Region: ${dimensions.width}x${dimensions.height}px at (${position.x}, ${position.y})`
      : `Element at (${position.x}, ${position.y})`

  return (
    <Card
      className="absolute z-20 w-72 shadow-lg"
      style={{
        left: position.x,
        top: position.y + (dimensions?.height ?? 0) + 8,
      }}
      role="dialog"
      aria-label="Add feedback comment"
    >
      <CardContent className="space-y-3 p-3">
        <div className="space-y-1">
          <p className="text-xs font-medium text-muted-foreground">
            {selectionLabel}
          </p>
          {capturedHtml && (
            <pre className="max-h-16 overflow-auto rounded bg-muted p-1.5 text-[10px] leading-tight text-muted-foreground">
              {capturedHtml.slice(0, 200)}
              {capturedHtml.length > 200 ? '...' : ''}
            </pre>
          )}
        </div>
        <Textarea
          ref={textareaRef}
          placeholder="Describe the feedback..."
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          onKeyDown={handleKeyDown}
          maxLength={2000}
          className="min-h-20 resize-none text-sm"
          aria-label="Feedback comment"
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onCancel}>
            Cancel
          </Button>
          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={!comment.trim()}
          >
            Add Comment
          </Button>
        </div>
        <p className="text-[10px] text-muted-foreground">
          Ctrl+Enter to submit
        </p>
      </CardContent>
    </Card>
  )
}
