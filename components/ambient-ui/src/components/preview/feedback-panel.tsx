'use client'

import { useCallback, useEffect, useState } from 'react'
import {
  Trash2,
  PanelRightClose,
  PanelRightOpen,
  Send,
  MapPin,
  Square,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import type { FeedbackItem } from '@/domain/types'

export type FeedbackPanelProps = {
  pendingItems: FeedbackItem[]
  sentItems: FeedbackItem[]
  onRemoveItem: (id: string) => void
  onUpdateComment: (id: string, comment: string) => void
  onSendAll: () => void
}

export function FeedbackPanel({
  pendingItems,
  sentItems,
  onRemoveItem,
  onUpdateComment,
  onSendAll,
}: FeedbackPanelProps) {
  const [collapsed, setCollapsed] = useState(false)
  const [confirming, setConfirming] = useState(false)

  const totalPending = pendingItems.length

  // Reset confirmation when items change (user adds/removes feedback)
  useEffect(() => {
    setConfirming(false)
  }, [totalPending])

  if (collapsed) {
    return (
      <div className="flex flex-col items-center border-l border-border bg-background py-2">
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => setCollapsed(false)}
          aria-label="Expand feedback panel"
        >
          <PanelRightOpen className="size-4" />
        </Button>
        {totalPending > 0 && (
          <span className="mt-1 flex size-5 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
            {totalPending}
          </span>
        )}
      </div>
    )
  }

  return (
    <div
      className="flex w-80 shrink-0 flex-col border-l border-border bg-background"
      role="complementary"
      aria-label="Feedback panel"
    >
      {/* Panel header */}
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-sm font-medium">
          Feedback{totalPending > 0 ? ` (${totalPending})` : ''}
        </span>
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => setCollapsed(true)}
          aria-label="Collapse feedback panel"
        >
          <PanelRightClose className="size-4" />
        </Button>
      </div>

      {/* Send all button with inline confirmation */}
      {totalPending > 0 && (
        <div className="border-b border-border px-3 py-2">
          {confirming ? (
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">
                Confirm send?
              </span>
              <div className="ml-auto flex gap-1">
                <Button
                  size="sm"
                  onClick={() => {
                    onSendAll()
                    setConfirming(false)
                  }}
                >
                  Confirm ({totalPending})
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setConfirming(false)}
                >
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <Button
              className="w-full gap-2"
              size="sm"
              onClick={() => setConfirming(true)}
            >
              <Send className="size-3.5" />
              Send All Feedback ({totalPending})
            </Button>
          )}
        </div>
      )}

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto">
        {/* Pending items */}
        {pendingItems.length > 0 && (
          <div className="px-3 py-2">
            <p className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Pending
            </p>
            <div className="space-y-2">
              {pendingItems.map((item) => (
                <PendingFeedbackCard
                  key={item.id}
                  item={item}
                  onRemove={() => onRemoveItem(item.id)}
                  onUpdateComment={(comment) =>
                    onUpdateComment(item.id, comment)
                  }
                />
              ))}
            </div>
          </div>
        )}

        {/* Sent items */}
        {sentItems.length > 0 && (
          <div className="px-3 py-2">
            <p className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Sent
            </p>
            <div className="space-y-2">
              {sentItems.map((item) => (
                <SentFeedbackCard key={item.id} item={item} />
              ))}
            </div>
          </div>
        )}

        {/* Empty state */}
        {pendingItems.length === 0 && sentItems.length === 0 && (
          <div className="flex flex-col items-center justify-center px-4 py-8 text-center">
            <p className="text-sm text-muted-foreground">No feedback yet</p>
            <p className="mt-1 text-xs text-muted-foreground">
              Click or drag on the preview to add feedback
            </p>
          </div>
        )}
      </div>
    </div>
  )
}

function PendingFeedbackCard({
  item,
  onRemove,
  onUpdateComment,
}: {
  item: FeedbackItem
  onRemove: () => void
  onUpdateComment: (comment: string) => void
}) {
  const [editing, setEditing] = useState(false)
  const [editValue, setEditValue] = useState(item.comment)

  const handleSaveEdit = useCallback(() => {
    onUpdateComment(editValue)
    setEditing(false)
  }, [editValue, onUpdateComment])

  const icon =
    item.type === 'region' ? (
      <Square className="size-3 text-blue-500" />
    ) : (
      <MapPin className="size-3 text-blue-500" />
    )

  const locationLabel =
    item.type === 'region' && item.dimensions
      ? `${item.dimensions.width}x${item.dimensions.height}px`
      : `(${item.position.x}, ${item.position.y})`

  return (
    <div className="rounded-md border border-border bg-card p-2.5">
      <div className="mb-1.5 flex items-start justify-between">
        <div className="flex items-center gap-1.5">
          {icon}
          <span className="text-[10px] text-muted-foreground">
            {locationLabel}
          </span>
        </div>
        <Button
          variant="ghost"
          size="icon-xs"
          onClick={onRemove}
          aria-label={`Remove feedback: ${item.comment.slice(0, 30)}`}
        >
          <Trash2 className="size-3" />
        </Button>
      </div>

      {editing ? (
        <div className="space-y-1.5">
          <Textarea
            value={editValue}
            onChange={(e) => setEditValue(e.target.value)}
            className="min-h-12 resize-none text-xs"
            aria-label="Edit feedback comment"
          />
          <div className="flex justify-end gap-1">
            <Button
              variant="ghost"
              size="xs"
              onClick={() => {
                setEditValue(item.comment)
                setEditing(false)
              }}
            >
              Cancel
            </Button>
            <Button size="xs" onClick={handleSaveEdit}>
              Save
            </Button>
          </div>
        </div>
      ) : (
        <button
          type="button"
          className="w-full cursor-pointer text-left text-xs text-foreground hover:text-foreground/80"
          onClick={() => setEditing(true)}
          aria-label="Edit comment"
        >
          {item.comment}
        </button>
      )}
    </div>
  )
}

function SentFeedbackCard({ item }: { item: FeedbackItem }) {
  const icon =
    item.type === 'region' ? (
      <Square className="size-3 text-muted-foreground" />
    ) : (
      <MapPin className="size-3 text-muted-foreground" />
    )

  const locationLabel =
    item.type === 'region' && item.dimensions
      ? `${item.dimensions.width}x${item.dimensions.height}px`
      : `(${item.position.x}, ${item.position.y})`

  return (
    <div className="rounded-md border border-border/50 bg-muted/30 p-2.5 opacity-60">
      <div className="mb-1 flex items-center gap-1.5">
        {icon}
        <span className="text-[10px] text-muted-foreground">
          {locationLabel}
        </span>
        <span className="ml-auto text-[10px] text-muted-foreground">Sent</span>
      </div>
      <p className="text-xs text-muted-foreground">{item.comment}</p>
    </div>
  )
}
