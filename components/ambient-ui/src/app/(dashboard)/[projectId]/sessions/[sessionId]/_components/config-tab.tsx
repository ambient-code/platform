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
import { getRegisteredAnnotation } from '@/domain/annotations'
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
  Eye,
  EyeOff,
} from 'lucide-react'
import { MetaRow, NoValue } from './meta-row'

const ICON_MAP: Record<string, LucideIcon> = {
  pin: Pin, tag: Tag, ticket: Ticket, layers: Layers, play: Play, bot: Bot,
  siren: Siren, user: User, 'dollar-sign': DollarSign,
  'git-pull-request': GitPullRequest, 'git-branch': GitBranch,
  'folder-git-2': FolderGit2, 'external-link': ExternalLink,
  'message-circle': MessageCircle, 'alert-triangle': AlertTriangle,
}

const PROMPT_TRUNCATE_LENGTH = 200

const SECRET_PATTERNS = /SECRET|TOKEN|PASSWORD|KEY|API|CREDENTIAL/i

function isSecretKey(key: string): boolean {
  return SECRET_PATTERNS.test(key)
}

function isClickableValue(value: string): boolean {
  return /^https?:\/\//.test(value)
}

function SecretValue({ value }: { value: string }) {
  const [revealed, setRevealed] = useState(false)

  return (
    <span className="inline-flex items-center gap-1.5">
      <span className="font-mono text-xs">
        {revealed ? value : '••••••••'}
      </span>
      <button
        type="button"
        className="inline-flex items-center text-muted-foreground hover:text-foreground"
        onClick={() => setRevealed((prev) => !prev)}
        aria-label={revealed ? 'Hide secret value' : 'Reveal secret value'}
      >
        {revealed ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
      </button>
    </span>
  )
}

export function ConfigTab({ session }: { session: DomainSession }) {
  const [promptExpanded, setPromptExpanded] = useState(false)

  const envEntries = Object.entries(session.environmentVariables)
  const annotationEntries = Object.entries(session.annotations)
  const labelEntries = Object.entries(session.labels)

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
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
            <MetaRow label="Model" value={session.model ?? <NoValue />} />
            <MetaRow label="Temperature" value={session.temperature != null ? String(session.temperature) : <NoValue />} />
            <MetaRow label="Max Tokens" value={session.maxTokens != null ? String(session.maxTokens) : <NoValue />} />
            <MetaRow label="Timeout" value={session.timeout != null ? `${session.timeout}s` : <NoValue />} />
            <MetaRow
              label="Workflow ID"
              value={
                session.workflowId
                  ? <span title="Workflow ID" className="font-mono text-xs">{session.workflowId}</span>
                  : <NoValue />
              }
            />
            {session.sdkRestartCount > 0 && (
              <MetaRow label="Agent Restarts" value={String(session.sdkRestartCount)} />
            )}
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
                  : `Show more (${session.prompt!.length.toLocaleString()} chars)`}
              </button>
            )}
          </CardContent>
        </Card>
      )}

      {envEntries.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              Environment Variables ({envEntries.length})
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
                {envEntries.map(([key, value]) => (
                  <TableRow key={key}>
                    <TableCell className="font-mono text-xs">{key}</TableCell>
                    <TableCell className="text-sm">
                      {isSecretKey(key) ? <SecretValue value={value} /> : value}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
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
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Key</TableHead>
                  <TableHead>Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {labelEntries.map(([key, value]) => (
                  <TableRow key={key}>
                    <TableCell className="font-mono text-xs">{key}</TableCell>
                    <TableCell className="text-sm">{value}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
