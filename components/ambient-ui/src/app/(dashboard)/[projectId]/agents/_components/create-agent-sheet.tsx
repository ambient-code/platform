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
import { useCreateAgent } from '@/queries/use-agents'
import type { DomainAgentCreateRequest } from '@/domain/types'

const MODEL_OPTIONS = [
  'claude-sonnet-4-20250514',
  'claude-opus-4-20250514',
  'claude-haiku-35-20241022',
] as const

export function CreateAgentSheet({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const router = useRouter()
  const { projectId } = useParams<{ projectId: string }>()
  const createAgent = useCreateAgent()

  const [name, setName] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [model, setModel] = useState('')
  const [prompt, setPrompt] = useState('')
  const [repoUrl, setRepoUrl] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)

  function resetForm() {
    setName('')
    setDisplayName('')
    setModel('')
    setPrompt('')
    setRepoUrl('')
    setDescription('')
    setError(null)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    if (!name.trim()) {
      setError('Name is required.')
      return
    }

    const request: DomainAgentCreateRequest = {
      name: name.trim(),
      projectId,
    }

    if (displayName.trim()) request.displayName = displayName.trim()
    if (model) request.model = model
    if (prompt.trim()) request.prompt = prompt.trim()
    if (repoUrl.trim()) request.repoUrl = repoUrl.trim()
    if (description.trim()) request.description = description.trim()

    try {
      const agent = await createAgent.mutateAsync({ projectId, request })
      resetForm()
      onOpenChange(false)
      router.push(`/${projectId}/agents/${agent.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create agent.')
    }
  }

  return (
    <Sheet open={open} onOpenChange={(v) => { if (!v) resetForm(); onOpenChange(v) }}>
      <SheetContent side="right" className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>New Agent</SheetTitle>
          <SheetDescription>
            Create a new agent definition in this project.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4 px-4 pb-4">
          <div className="space-y-1.5">
            <label htmlFor="agent-name" className="text-sm font-medium">
              Name <span className="text-destructive">*</span>
            </label>
            <Input
              id="agent-name"
              placeholder="my-agent"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="agent-display-name" className="text-sm font-medium">
              Display Name
            </label>
            <Input
              id="agent-display-name"
              placeholder="My Agent"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="agent-model" className="text-sm font-medium">
              Model
            </label>
            <Select value={model} onValueChange={setModel}>
              <SelectTrigger id="agent-model" className="w-full">
                <SelectValue placeholder="Select a model (optional)" />
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

          <div className="space-y-1.5">
            <label htmlFor="agent-prompt" className="text-sm font-medium">
              Prompt
            </label>
            <Textarea
              id="agent-prompt"
              placeholder="System prompt for the agent..."
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              className="min-h-24 font-mono text-sm"
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="agent-repo-url" className="text-sm font-medium">
              Repository URL
            </label>
            <Input
              id="agent-repo-url"
              placeholder="https://github.com/org/repo"
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="agent-description" className="text-sm font-medium">
              Description
            </label>
            <Textarea
              id="agent-description"
              placeholder="What does this agent do?"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="min-h-20"
            />
          </div>

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
              disabled={createAgent.isPending || !name.trim()}
            >
              {createAgent.isPending ? 'Creating...' : 'Create Agent'}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
