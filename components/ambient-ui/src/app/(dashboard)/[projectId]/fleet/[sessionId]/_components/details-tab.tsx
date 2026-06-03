'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { DomainSession } from '@/domain/types'
import { getRegisteredAnnotations } from '@/domain/annotations'
import { cn } from '@/lib/utils'
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

export function DetailsTab({ session }: { session: DomainSession }) {
  const [promptExpanded, setPromptExpanded] = useState(false)

  const envEntries = Object.entries(session.environmentVariables)
  const annotationEntries = Object.entries(session.annotations)
  const labelEntries = Object.entries(session.labels)
  const registered = getRegisteredAnnotations(session.annotations)

  const promptNeedsTruncation =
    session.prompt != null && session.prompt.length > PROMPT_TRUNCATE_LENGTH
  const displayPrompt =
    session.prompt != null
      ? promptNeedsTruncation && !promptExpanded
        ? session.prompt.slice(0, PROMPT_TRUNCATE_LENGTH) + '…'
        : session.prompt
      : null

  return (
    <div className="space-y-6 pt-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm">
            <MetaRow label="Model" value={session.model ?? '—'} />
            <MetaRow label="Temperature" value={session.temperature != null ? String(session.temperature) : '—'} />
            <MetaRow label="Max Tokens" value={session.maxTokens != null ? String(session.maxTokens) : '—'} />
            <MetaRow label="Timeout" value={session.timeout != null ? `${session.timeout}s` : '—'} />
            <MetaRow label="Workflow ID" value={session.workflowId ?? '—'} mono />
            <MetaRow label="SDK Restart Count" value={String(session.sdkRestartCount)} />
          </dl>
        </CardContent>
      </Card>

      {displayPrompt != null && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Prompt</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="whitespace-pre-wrap text-sm font-mono">
              {displayPrompt}
            </pre>
            {promptNeedsTruncation && (
              <button
                type="button"
                className="mt-2 text-xs text-muted-foreground underline hover:text-foreground"
                onClick={() => setPromptExpanded((prev) => !prev)}
              >
                {promptExpanded ? 'Show less' : 'Show more'}
              </button>
            )}
          </CardContent>
        </Card>
      )}

      {envEntries.length > 0 && (
        <KeyValueCard title="Environment Variables" entries={envEntries} />
      )}

      {registered.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Registered Annotations</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {registered.map(({ annotation, value }) => {
                const Icon = annotation.icon ? ICON_MAP[annotation.icon] : null
                const clickable = isClickableValue(value)
                return (
                  <div key={annotation.key} className="flex items-center gap-3 text-sm">
                    {Icon && <Icon className="size-4 shrink-0 text-muted-foreground" />}
                    <span className="text-muted-foreground shrink-0">
                      {annotation.label}
                    </span>
                    {clickable ? (
                      <a
                        href={value}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="truncate text-blue-500 underline hover:text-blue-400"
                      >
                        {value}
                      </a>
                    ) : (
                      <span className="truncate">{value}</span>
                    )}
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {annotationEntries.length > 0 && (
        <KeyValueCard title="Raw Annotations" entries={annotationEntries} />
      )}

      {labelEntries.length > 0 && (
        <KeyValueCard title="Labels" entries={labelEntries} />
      )}
    </div>
  )
}

function MetaRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={cn('mt-0.5', mono && 'font-mono text-xs')}>{value}</dd>
    </div>
  )
}

function KeyValueCard({ title, entries }: { title: string; entries: [string, string][] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
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
            {entries.map(([key, value]) => (
              <TableRow key={key}>
                <TableCell className="font-mono text-xs">{key}</TableCell>
                <TableCell className="text-sm">{value}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}
