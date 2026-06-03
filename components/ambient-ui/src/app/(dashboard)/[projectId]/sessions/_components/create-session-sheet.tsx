'use client'

import { useState } from 'react'
import { useRouter, useParams } from 'next/navigation'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCreateSession } from '@/queries/use-sessions'
import { useAgents } from '@/queries/use-agents'
import type { DomainSessionCreateRequest } from '@/domain/types'

const MODEL_OPTIONS = [
  'claude-sonnet-4-20250514',
  'claude-opus-4-20250514',
  'claude-haiku-35-20241022',
] as const

export function CreateSessionSheet({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()
  const { projectId } = useParams<{ projectId: string }>()
  const createSession = useCreateSession()
  const { data: agentsData } = useAgents(projectId, { size: 100 })

  const [name, setName] = useState('')
  const [agentId, setAgentId] = useState('')
  const [prompt, setPrompt] = useState('')
  const [model, setModel] = useState('')
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [temperature, setTemperature] = useState('')
  const [maxTokens, setMaxTokens] = useState('')
  const [timeout, setTimeout] = useState('')
  const [error, setError] = useState<string | null>(null)

  const agents = agentsData?.items ?? []

  function resetForm() {
    setName('')
    setAgentId('')
    setPrompt('')
    setModel('')
    setShowAdvanced(false)
    setTemperature('')
    setMaxTokens('')
    setTimeout('')
    setError(null)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    if (!name.trim()) {
      setError('Name is required.')
      return
    }

    const request: DomainSessionCreateRequest = {
      name: name.trim(),
      projectId,
    }

    if (agentId) request.agentId = agentId
    if (prompt.trim()) request.prompt = prompt.trim()
    if (model) request.model = model

    if (showAdvanced) {
      const tempVal = parseFloat(temperature)
      if (!isNaN(tempVal)) request.temperature = tempVal

      const maxTokVal = parseInt(maxTokens, 10)
      if (!isNaN(maxTokVal) && maxTokVal > 0) request.maxTokens = maxTokVal

      const timeoutVal = parseInt(timeout, 10)
      if (!isNaN(timeoutVal) && timeoutVal > 0) request.timeout = timeoutVal
    }

    try {
      const session = await createSession.mutateAsync(request)
      resetForm()
      onOpenChange(false)
      router.push(`/${projectId}/sessions/${session.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create session.')
    }
  }

  return (
    <Sheet open={open} onOpenChange={(v) => { if (!v) resetForm(); onOpenChange(v) }}>
      <SheetContent side="right" className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>New Session</SheetTitle>
          <SheetDescription>
            Create a new agentic session in this project.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4 px-4 pb-4">
          <div className="space-y-1.5">
            <label htmlFor="session-name" className="text-sm font-medium">
              Name <span className="text-destructive">*</span>
            </label>
            <Input
              id="session-name"
              placeholder="my-session"
              value={name}
              onChange={e => setName(e.target.value)}
              required
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="session-agent" className="text-sm font-medium">
              Agent
            </label>
            <Select value={agentId} onValueChange={setAgentId}>
              <SelectTrigger id="session-agent" className="w-full">
                <SelectValue placeholder="Select an agent (optional)" />
              </SelectTrigger>
              <SelectContent>
                {agents.map(agent => (
                  <SelectItem key={agent.id} value={agent.id}>
                    {agent.displayName ?? agent.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <label htmlFor="session-prompt" className="text-sm font-medium">
              Prompt
            </label>
            <Textarea
              id="session-prompt"
              placeholder="Describe what this session should do..."
              value={prompt}
              onChange={e => setPrompt(e.target.value)}
              className="min-h-24"
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="session-model" className="text-sm font-medium">
              Model
            </label>
            <Select value={model} onValueChange={setModel}>
              <SelectTrigger id="session-model" className="w-full">
                <SelectValue placeholder="Select a model (optional)" />
              </SelectTrigger>
              <SelectContent>
                {MODEL_OPTIONS.map(m => (
                  <SelectItem key={m} value={m}>
                    {m}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <button
            type="button"
            className="text-sm text-muted-foreground underline hover:text-foreground text-left"
            onClick={() => setShowAdvanced(prev => !prev)}
          >
            {showAdvanced ? 'Hide advanced settings' : 'Show advanced settings'}
          </button>

          {showAdvanced && (
            <div className="space-y-4 rounded-md border p-4">
              <div className="space-y-1.5">
                <label htmlFor="session-temperature" className="text-sm font-medium">
                  Temperature
                </label>
                <Input
                  id="session-temperature"
                  type="number"
                  step="0.1"
                  min="0"
                  max="2"
                  placeholder="0.7"
                  value={temperature}
                  onChange={e => setTemperature(e.target.value)}
                />
              </div>

              <div className="space-y-1.5">
                <label htmlFor="session-max-tokens" className="text-sm font-medium">
                  Max Tokens
                </label>
                <Input
                  id="session-max-tokens"
                  type="number"
                  min="1"
                  placeholder="4096"
                  value={maxTokens}
                  onChange={e => setMaxTokens(e.target.value)}
                />
              </div>

              <div className="space-y-1.5">
                <label htmlFor="session-timeout" className="text-sm font-medium">
                  Timeout (seconds)
                </label>
                <Input
                  id="session-timeout"
                  type="number"
                  min="1"
                  placeholder="3600"
                  value={timeout}
                  onChange={e => setTimeout(e.target.value)}
                />
              </div>
            </div>
          )}

          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}

          <SheetFooter className="px-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => { resetForm(); onOpenChange(false) }}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createSession.isPending || !name.trim()}
            >
              {createSession.isPending ? 'Creating...' : 'Create Session'}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
