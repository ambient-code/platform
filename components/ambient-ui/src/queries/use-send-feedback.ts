'use client'

import { useMutation } from '@tanstack/react-query'
import type { FeedbackBatch, FeedbackItem } from '@/domain/types'
import type { SessionMessagesPort } from '@/ports/session-messages'
import { createSessionMessagesAdapterWithFetch } from '@/adapters/session-messages'
import { domainProbe } from '@/lib/observability'
import { SESSION_MESSAGE_EVENTS } from '@/domain/events'

function formatFeedbackPayload(batch: FeedbackBatch): string {
  const header = `Visual feedback for preview: ${sanitizeForPayload(batch.previewUrl)}`
  const items = batch.items.map((item, i) => formatFeedbackItem(item, i + 1))
  return `${header}\n\n${items.join('\n\n')}`
}

function sanitizeForPayload(value: string): string {
  return value.replace(/`/g, '\\`').replace(/\r?\n/g, ' ')
}

function formatFeedbackItem(item: FeedbackItem, index: number): string {
  const lines: string[] = []
  lines.push(`--- Feedback #${index} ---`)
  lines.push(`Comment: ${item.comment}`)
  lines.push(`Type: ${item.type}`)
  lines.push(`Position: (${item.position.x}, ${item.position.y})`)

  if (item.dimensions) {
    lines.push(`Region: ${item.dimensions.width}x${item.dimensions.height}px`)
  }

  lines.push(`Viewport: ${item.viewportWidth}x${item.viewportHeight} (${item.deviceSize})`)

  if (item.capturedHtml) {
    const escaped = sanitizeForPayload(item.capturedHtml)
    lines.push(`Element HTML:\n\`\`\`html\n${escaped}\n\`\`\``)
  }

  return lines.join('\n')
}

export function useSendFeedback(port?: SessionMessagesPort) {
  const adapter = port ?? createSessionMessagesAdapterWithFetch()

  return useMutation({
    mutationFn: async (batch: FeedbackBatch) => {
      const payload = formatFeedbackPayload(batch)
      return adapter.send(batch.sessionId, {
        eventType: SESSION_MESSAGE_EVENTS.userFeedback,
        payload,
      })
    },
    onSuccess: (_data, batch) => {
      domainProbe.feedbackSent({
        sessionId: batch.sessionId,
        itemCount: batch.items.length,
        previewUrl: batch.previewUrl,
      })
    },
    onError: (error, batch) => {
      domainProbe.feedbackDeliveryFailed({
        sessionId: batch.sessionId,
        error: error instanceof Error ? error.message : String(error),
      })
    },
  })
}
