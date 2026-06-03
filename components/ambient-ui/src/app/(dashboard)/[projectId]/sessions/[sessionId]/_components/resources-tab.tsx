import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { EmptyState } from '@/components/empty-state'
import type { DomainSession, DomainRepo, DomainReconciledRepo, ReconciledRepoStatus } from '@/domain/types'
import { formatAbsoluteTime } from '@/lib/format-timestamp'
import { cn } from '@/lib/utils'
import { FolderGit2 } from 'lucide-react'
import { NoValue } from './meta-row'

const STATUS_CLASSES: Record<ReconciledRepoStatus, string> = {
  Ready: 'bg-status-success text-status-success-foreground border-status-success-border',
  Cloning: 'bg-status-warning text-status-warning-foreground border-status-warning-border',
  Failed: 'bg-status-error text-status-error-foreground border-status-error-border',
}

type MergedRepo = {
  url: string
  name: string
  branch: string | null
  status: ReconciledRepoStatus | null
  clonedAt: string | null
}

function mergeRepos(
  repos: DomainRepo[],
  reconciledRepos: DomainReconciledRepo[],
): MergedRepo[] {
  const reconciledByUrl = new Map(
    reconciledRepos.map(r => [r.url, r]),
  )

  const seen = new Set<string>()
  const result: MergedRepo[] = []

  for (const repo of repos) {
    seen.add(repo.url)
    const reconciled = reconciledByUrl.get(repo.url)
    result.push({
      url: repo.url,
      name: reconciled?.name ?? repo.name ?? baseNameFromUrl(repo.url),
      branch: reconciled?.currentActiveBranch ?? repo.branch ?? null,
      status: reconciled?.status ?? null,
      clonedAt: reconciled?.clonedAt ?? null,
    })
  }

  for (const reconciled of reconciledRepos) {
    if (!seen.has(reconciled.url)) {
      result.push({
        url: reconciled.url,
        name: reconciled.name ?? baseNameFromUrl(reconciled.url),
        branch: reconciled.currentActiveBranch ?? null,
        status: reconciled.status,
        clonedAt: reconciled.clonedAt,
      })
    }
  }

  return result
}

function baseNameFromUrl(url: string): string {
  const segments = url.replace(/\.git$/, '').split('/')
  return segments[segments.length - 1] || url
}

export function ResourcesTab({ session }: { session: DomainSession }) {
  const merged = mergeRepos(session.repos, session.reconciledRepos)
  const hasRepos = merged.length > 0

  if (!hasRepos) {
    return (
      <div className="pt-4">
        <EmptyState
          icon={FolderGit2}
          title="No resources attached"
          description="This session has no repositories configured."
        />
      </div>
    )
  }

  return (
    <div className="space-y-6 pt-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            <FolderGit2 className="mr-2 inline-block size-4" />
            Repositories ({merged.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>URL</TableHead>
                <TableHead>Branch</TableHead>
                <TableHead>Clone Status</TableHead>
                <TableHead>Cloned At</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {merged.map(repo => (
                <TableRow key={repo.url}>
                  <TableCell className="font-medium">{repo.name}</TableCell>
                  <TableCell className="max-w-[300px] truncate font-mono text-xs">
                    <a
                      href={repo.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      title={repo.url}
                      className="text-link hover:text-link-hover"
                    >
                      {repo.url}
                    </a>
                  </TableCell>
                  <TableCell className="text-sm">
                    {repo.branch ?? <NoValue />}
                  </TableCell>
                  <TableCell>
                    {repo.status ? (
                      <Badge
                        variant="outline"
                        className={cn('font-medium', STATUS_CLASSES[repo.status])}
                      >
                        {repo.status}
                      </Badge>
                    ) : (
                      <NoValue />
                    )}
                  </TableCell>
                  <TableCell className="text-sm">
                    {repo.clonedAt ? formatAbsoluteTime(repo.clonedAt) : <NoValue />}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
