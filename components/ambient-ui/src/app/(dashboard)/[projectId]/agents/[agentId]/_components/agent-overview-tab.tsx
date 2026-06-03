'use client'

import { useState, useEffect, useCallback } from 'react'
import type { LucideIcon } from 'lucide-react'
import {
  Pin, Tag, Ticket, GitPullRequest, GitBranch, FolderGit2,
  Layers, ExternalLink, MessageCircle, User, Play,
  DollarSign, Siren, Bot, AlertTriangle, Info,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getRegisteredAnnotation } from '@/domain/annotations'
import type { DomainAgent } from '@/domain/types'
import type { AgentLifecycle } from '../../_components/lifecycle-badge'
import { useUpdateAgent } from '@/queries/use-agents'
import { MODEL_OPTIONS } from '@/domain/models'

const ICON_MAP: Record<string, LucideIcon> = {
  pin: Pin, tag: Tag, ticket: Ticket, layers: Layers, play: Play, bot: Bot,
  siren: Siren, user: User, 'dollar-sign': DollarSign,
  'git-pull-request': GitPullRequest, 'git-branch': GitBranch,
  'folder-git-2': FolderGit2, 'external-link': ExternalLink,
  'message-circle': MessageCircle, 'alert-triangle': AlertTriangle,
}

function isClickableValue(value: string): boolean {
  return /^https?:\/\//.test(value)
}

export function AgentOverviewTab({
  agent,
  lifecycle,
}: {
  agent: DomainAgent
  lifecycle: AgentLifecycle
}) {
  const isGitOps = lifecycle === 'gitops'
  const updateAgent = useUpdateAgent()

  const [displayName, setDisplayName] = useState(agent.displayName ?? '')
  const [model, setModel] = useState(agent.model ?? '')
  const [prompt, setPrompt] = useState(agent.prompt ?? '')
  const [repoUrl, setRepoUrl] = useState(agent.repoUrl ?? '')
  const [description, setDescription] = useState(agent.description ?? '')
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saveSuccess, setSaveSuccess] = useState(false)

  useEffect(() => {
    setDisplayName(agent.displayName ?? '')
    setModel(agent.model ?? '')
    setPrompt(agent.prompt ?? '')
    setRepoUrl(agent.repoUrl ?? '')
    setDescription(agent.description ?? '')
  }, [agent])

  const handleSave = useCallback(async () => {
    setSaveError(null)
    setSaveSuccess(false)
    try {
      await updateAgent.mutateAsync({
        projectId: agent.projectId ?? '',
        agentId: agent.id,
        request: {
          displayName: displayName || undefined,
          model: model || undefined,
          prompt: prompt || undefined,
          repoUrl: repoUrl || undefined,
          description: description || undefined,
        },
      })
      setSaveSuccess(true)
      globalThis.setTimeout(() => setSaveSuccess(false), 3000)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save changes.')
    }
  }, [updateAgent, agent.id, displayName, model, prompt, repoUrl, description])

  const annotationEntries = Object.entries(agent.annotations)

  return (
    <div className="space-y-6 pt-4">
      {isGitOps && (
        <div className="flex items-start gap-3 rounded-md border border-muted bg-muted/50 p-4">
          <Info className="size-5 shrink-0 text-muted-foreground mt-0.5" />
          <div>
            <p className="text-sm font-medium">GitOps-managed agent</p>
            <p className="text-sm text-muted-foreground">
              This agent is managed via GitOps. Edits here will not persist.
            </p>
          </div>
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Agent Configuration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="agent-display-name" className="text-sm font-medium">
              Display Name
            </label>
            <Input
              id="agent-display-name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Human-readable name"
              disabled={isGitOps}
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="agent-model" className="text-sm font-medium">
              Model
            </label>
            <Select value={model} onValueChange={setModel} disabled={isGitOps}>
              <SelectTrigger id="agent-model" className="w-full">
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

          <div className="space-y-1.5">
            <label htmlFor="agent-repo-url" className="text-sm font-medium">
              Repository URL
            </label>
            <Input
              id="agent-repo-url"
              value={repoUrl}
              onChange={(e) => setRepoUrl(e.target.value)}
              placeholder="https://github.com/org/repo"
              disabled={isGitOps}
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="agent-description" className="text-sm font-medium">
              Description
            </label>
            <Textarea
              id="agent-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What does this agent do?"
              className="min-h-20"
              disabled={isGitOps}
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="agent-prompt" className="text-sm font-medium">
              Prompt
            </label>
            <Textarea
              id="agent-prompt"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="System prompt for the agent..."
              className="min-h-40 font-mono text-sm"
              disabled={isGitOps}
            />
          </div>

          {!isGitOps && (
            <div className="flex items-center gap-3 pt-2">
              <Button
                onClick={handleSave}
                disabled={updateAgent.isPending}
              >
                {updateAgent.isPending ? 'Saving...' : 'Save Changes'}
              </Button>
              {saveSuccess && (
                <span className="text-sm text-success-foreground">Changes saved.</span>
              )}
              {saveError && (
                <span className="text-sm text-destructive">{saveError}</span>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {annotationEntries.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              Annotations ({annotationEntries.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Key</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {annotationEntries.map(([key, value]) => {
                  const registered = getRegisteredAnnotation(key)
                  const Icon = registered?.icon ? ICON_MAP[registered.icon] : null
                  const clickable = isClickableValue(value)
                  return (
                    <TableRow key={key}>
                      <TableCell className="font-mono text-xs">
                        <span className="inline-flex items-center gap-1.5">
                          {Icon && <Icon className="size-3.5 shrink-0 text-muted-foreground" />}
                          {registered ? registered.label : key}
                        </span>
                      </TableCell>
                      <TableCell className="text-sm">
                        {clickable ? (
                          <a
                            href={value}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="truncate text-link underline hover:text-link-hover"
                          >
                            {value}
                          </a>
                        ) : (
                          value
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {Object.keys(agent.labels).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              Labels ({Object.keys(agent.labels).length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {Object.entries(agent.labels).map(([key, value]) => (
                <Badge key={key} variant="secondary" className="text-xs">
                  {key}: {value}
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
