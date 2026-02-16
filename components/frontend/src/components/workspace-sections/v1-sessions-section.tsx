'use client';

import { useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { RefreshCw, MoreVertical, Square, Brain, Search, ChevronLeft, ChevronRight, Play, Plus } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { EmptyState } from '@/components/empty-state';
import { SessionPhaseBadge } from '@/components/status-badge';
import { V1CreateSessionDialog } from '@/components/v1-create-session-dialog';

import { useV1Sessions, useV1StopSession, useV1StartSession } from '@/services/queries/v1';
import { successToast, errorToast } from '@/hooks/use-toast';
import { useDebounce } from '@/hooks/use-debounce';
import type { Session } from '@ambient-platform/sdk';

type V1SessionsSectionProps = {
  projectName: string;
};

export function V1SessionsSection({ projectName }: V1SessionsSectionProps) {
  const [searchInput, setSearchInput] = useState('');
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const debouncedSearch = useDebounce(searchInput, 300);

  const {
    data: listResponse,
    isFetching,
    refetch,
  } = useV1Sessions(projectName, {
    page,
    size: pageSize,
    search: debouncedSearch || undefined,
  });

  const sessions = listResponse?.items ?? [];
  const total = listResponse?.total ?? 0;
  const totalPages = Math.ceil(total / pageSize);

  const stopMutation = useV1StopSession(projectName);
  const startMutation = useV1StartSession(projectName);

  const handleStop = (id: string) => {
    stopMutation.mutate(id, {
      onSuccess: () => successToast('Session stopped'),
      onError: (err) => errorToast(err instanceof Error ? err.message : 'Failed to stop session'),
    });
  };

  const handleStart = (id: string) => {
    startMutation.mutate(id, {
      onSuccess: () => successToast('Session started'),
      onError: (err) => errorToast(err instanceof Error ? err.message : 'Failed to start session'),
    });
  };

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>Sessions (API Server)</CardTitle>
            <CardDescription>Sessions from the ambient-api-server</CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => refetch()} disabled={isFetching}>
              <RefreshCw className={`w-4 h-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <V1CreateSessionDialog
              projectName={projectName}
              onSuccess={() => refetch()}
              trigger={
                <Button>
                  <Plus className="w-4 h-4 mr-2" />
                  New Session
                </Button>
              }
            />
          </div>
        </div>
        <div className="relative mt-4 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search sessions..."
            value={searchInput}
            onChange={(e) => { setSearchInput(e.target.value); setPage(1); }}
            className="pl-9"
          />
        </div>
      </CardHeader>
      <CardContent>
        {sessions.length === 0 && !debouncedSearch ? (
          <EmptyState
            icon={Brain}
            title="No sessions found"
            description="No sessions exist on the API server yet"
          />
        ) : sessions.length === 0 && debouncedSearch ? (
          <EmptyState
            icon={Search}
            title="No matching sessions"
            description={`No sessions found matching "${debouncedSearch}"`}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[180px]">Name</TableHead>
                    <TableHead>Phase</TableHead>
                    <TableHead>Mode</TableHead>
                    <TableHead className="hidden md:table-cell">Model</TableHead>
                    <TableHead className="hidden lg:table-cell">Created</TableHead>
                    <TableHead className="w-[50px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sessions.map((session: Session) => (
                    <TableRow key={session.id}>
                      <TableCell className="font-medium min-w-[180px]">
                        <div>
                          <div className="font-medium">{session.name}</div>
                          <div className="text-xs text-muted-foreground font-normal">{session.id}</div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <SessionPhaseBadge phase={session.phase || 'Pending'} />
                      </TableCell>
                      <TableCell>
                        <span className="text-xs px-2 py-1 rounded border bg-muted/50">
                          {session.interactive ? 'Interactive' : 'Headless'}
                        </span>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <span className="text-sm text-muted-foreground truncate max-w-[120px] block">
                          {session.llm_model || '—'}
                        </span>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {session.created_at &&
                          formatDistanceToNow(new Date(session.created_at), { addSuffix: true })}
                      </TableCell>
                      <TableCell>
                        <V1SessionActions
                          session={session}
                          onStop={handleStop}
                          onStart={handleStart}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {totalPages > 1 && (
              <div className="flex items-center justify-between pt-4 border-t mt-4">
                <div className="text-sm text-muted-foreground">
                  Page {page} of {totalPages} ({total} total)
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page <= 1 || isFetching}
                  >
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => p + 1)}
                    disabled={page >= totalPages || isFetching}
                  >
                    Next
                    <ChevronRight className="h-4 w-4 ml-1" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}

function V1SessionActions({
  session,
  onStop,
  onStart,
}: {
  session: Session;
  onStop: (id: string) => void;
  onStart: (id: string) => void;
}) {
  const phase = (session.phase || '').toLowerCase();
  const canStop = phase === 'pending' || phase === 'creating' || phase === 'running';
  const canStart = phase === 'completed' || phase === 'failed' || phase === 'stopped' || phase === 'error';

  if (!canStop && !canStart) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
          <MoreVertical className="h-4 w-4" />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {canStop && (
          <DropdownMenuItem onClick={() => onStop(session.id)} className="text-orange-600">
            <Square className="h-4 w-4 mr-2" />
            Stop
          </DropdownMenuItem>
        )}
        {canStart && (
          <DropdownMenuItem onClick={() => onStart(session.id)} className="text-green-600">
            <Play className="h-4 w-4 mr-2" />
            Start
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
