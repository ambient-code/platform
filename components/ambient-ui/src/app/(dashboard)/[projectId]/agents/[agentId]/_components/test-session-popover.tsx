'use client'

import { useState, useCallback } from 'react'
import { useParams } from 'next/navigation'
import { Play } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCreateSession } from '@/queries/use-sessions'
import type { DomainAgent } from '@/domain/types'
import { MODEL_OPTIONS } from '@/domain/models'

type TestSessionPopoverProps = {
  agent: DomainAgent
  onRunTest: (sessionId: string, sessionName: string) => void
}

export function TestSessionPopover({ agent, onRunTest }: TestSessionPopoverProps) {
  const { projectId } = useParams<{ projectId: string }>()
  const [open, setOpen] = useState(false)
  const [promptOverride, setPromptOverride] = useState('')
  const [model, setModel] = useState(agent.model ?? MODEL_OPTIONS[0])
  const createSession = useCreateSession()

  const handleRun = useCallback(() => {
    const sessionName = `test-${agent.name}-${Date.now()}`
    const prompt = promptOverride.trim() || agent.prompt || undefined

    createSession.mutate(
      {
        name: sessionName,
        projectId,
        agentId: agent.id,
        prompt,
        model,
        annotations: { 'ambient-code.io/ui/test-session': 'true' },
      },
      {
        onSuccess: (session) => {
          setOpen(false)
          setPromptOverride('')
          onRunTest(session.id, session.name)
        },
      },
    )
  }, [agent, projectId, promptOverride, model, createSession, onRunTest])

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" aria-label="Run test session">
          <Play className="size-4" />
          Run Test Session
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80" align="end">
        <div className="space-y-4">
          <div className="space-y-2">
            <h4 className="text-sm font-medium">Run Test Session</h4>
            <p className="text-xs text-muted-foreground">
              Run a quick test session with this agent.
            </p>
          </div>

          <div className="space-y-1.5">
            <label htmlFor="test-prompt" className="text-xs font-medium">
              Prompt Override
            </label>
            <Textarea
              id="test-prompt"
              value={promptOverride}
              onChange={(e) => setPromptOverride(e.target.value)}
              placeholder="Uses agent's prompt if empty"
              className="min-h-20 text-sm"
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="test-model" className="text-xs font-medium">
              Model
            </label>
            <Select value={model} onValueChange={setModel}>
              <SelectTrigger id="test-model" className="w-full text-sm">
                <SelectValue placeholder="Select a model" />
              </SelectTrigger>
              <SelectContent>
                {MODEL_OPTIONS.map((m) => (
                  <SelectItem key={m} value={m}>
                    {m}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Button
            onClick={handleRun}
            disabled={createSession.isPending}
            className="w-full"
            size="sm"
          >
            {createSession.isPending ? 'Starting...' : 'Run'}
          </Button>

          {createSession.isError && (
            <p className="text-xs text-destructive">
              Failed to create test session. Please try again.
            </p>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
