'use client'

import { useState } from 'react'
import Link from 'next/link'
import type { LucideIcon } from 'lucide-react'
import {
  Pin,
  Tag,
  Ticket,
  GitPullRequest,
  GitBranch,
  FolderGit2,
  Layers,
  ExternalLink,
  MessageCircle,
  User,
  Play,
  DollarSign,
  Siren,
  Bot,
  AlertTriangle,
} from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { MetaRow, NoValue } from '@/app/(dashboard)/[projectId]/sessions/[sessionId]/_components/meta-row'
import { getRegisteredAnnotation } from '@/domain/annotations'
import type { DomainAgent } from '@/domain/types'
import { formatRelativeTime } from '@/lib/format-timestamp'

const ICON_MAP: Record<string, LucideIcon> = {
  pin: Pin, tag: Tag, ticket: Ticket, layers: Layers, play: Play, bot: Bot,
  siren: Siren, user: User, 'dollar-sign': DollarSign,
  'git-pull-request': GitPullRequest, 'git-branch': GitBranch,
  'folder-git-2': FolderGit2, 'external-link': ExternalLink,
  'message-circle': MessageCircle, 'alert-triangle': AlertTriangle,
}

const PROMPT_TRUNCATE_LENGTH = 200

function isClickableValue(value: string): boolean {
  return /^https?:\/\//.test(value)
}

export function AgentDetailPanel({
  agent,
  projectId,
  onClose,
}: {
  agent: DomainAgent | null
  projectId: string
  onClose: () => void
}) {
  const [promptExpanded, setPromptExpanded] = useState(false)

  const isOpen = agent !== null

  const annotationEntries = agent ? Object.entries(agent.annotations) : []
  const labelEntries = agent ? Object.entries(agent.labels) : []

  const promptNeedsTruncation =
    agent?.prompt != null && agent.prompt.length > PROMPT_TRUNCATE_LENGTH
  const displayPrompt =
    agent?.prompt != null
      ? promptNeedsTruncation && !promptExpanded
        ? agent.prompt.slice(0, PROMPT_TRUNCATE_LENGTH) + '...'
        : agent.prompt
      : null

  return (
    <Sheet open={isOpen} onOpenChange={(open) => { if (!open) onClose() }}>
      <SheetContent side="right" className="sm:max-w-lg overflow-y-auto">
        {agent && (
          <>
            <SheetHeader>
              <SheetTitle>{agent.displayName ?? agent.name}</SheetTitle>
              {agent.displayName && (
                <SheetDescription className="font-mono text-xs">
                  {agent.name}
                </SheetDescription>
              )}
            </SheetHeader>

            <div className="space-y-6 px-4 pb-4">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Quick Info</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
                    <MetaRow label="Model" value={agent.model ?? <NoValue />} />
                    <MetaRow label="Owner" value={agent.ownerUserId ?? <NoValue />} />
                    <MetaRow
                      label="Current Session"
                      value={
                        agent.currentSessionId ? (
                          <Link
                            href={`/${projectId}/sessions/${agent.currentSessionId}`}
                            className="text-xs font-mono text-link underline hover:text-link-hover"
                            onClick={(e) => e.stopPropagation()}
                          >
                            {agent.currentSessionId}
                          </Link>
                        ) : (
                          <NoValue />
                        )
                      }
                    />
                    <MetaRow
                      label="Repository"
                      value={
                        agent.repoUrl ? (
                          <a
                            href={agent.repoUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs text-link underline hover:text-link-hover truncate"
                          >
                            {agent.repoUrl}
                          </a>
                        ) : (
                          <NoValue />
                        )
                      }
                    />
                    <MetaRow label="Workflow ID" value={agent.workflowId ? <span className="font-mono text-xs">{agent.workflowId}</span> : <NoValue />} />
                    <MetaRow label="Description" value={agent.description ?? <NoValue />} />
                    <MetaRow label="Created" value={agent.createdAt ? formatRelativeTime(agent.createdAt) : <NoValue />} />
                    <MetaRow label="Updated" value={agent.updatedAt ? formatRelativeTime(agent.updatedAt) : <NoValue />} />
                  </dl>
                </CardContent>
              </Card>

              {displayPrompt != null && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Prompt</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <pre className="whitespace-pre-wrap text-sm font-mono">{displayPrompt}</pre>
                    {promptNeedsTruncation && (
                      <button
                        type="button"
                        className="mt-2 py-1 text-sm text-muted-foreground underline hover:text-foreground"
                        onClick={() => setPromptExpanded((prev) => !prev)}
                      >
                        {promptExpanded
                          ? 'Show less'
                          : `Show more (${agent.prompt!.length.toLocaleString()} chars)`}
                      </button>
                    )}
                  </CardContent>
                </Card>
              )}

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

              {labelEntries.length > 0 && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">
                      Labels ({labelEntries.length})
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="flex flex-wrap gap-2">
                      {labelEntries.map(([key, value]) => (
                        <Badge key={key} variant="secondary" className="text-xs">
                          {key}: {value}
                        </Badge>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}
