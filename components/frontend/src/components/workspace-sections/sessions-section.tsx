'use client';

import { useState, useEffect } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { Plus, RefreshCw, MoreVertical, Square, Trash2, ArrowRight, Brain, Search, Pencil, Clock, Cpu, MessageSquare, NotepadText, User, ArrowUp, ArrowDown } from 'lucide-react';
import Link from 'next/link';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
  PaginationEllipsis,
} from '@/components/ui/pagination';
import { getPageNumbers } from '@/lib/pagination';
import { EmptyState } from '@/components/empty-state';
import { SessionStatusDot } from '@/components/session-status-dot';
import { AgentStatusIndicator } from '@/components/agent-status-indicator';
import { deriveAgentStatusFromPhase } from '@/hooks/use-agent-status';
import { EditSessionNameDialog } from '@/components/edit-session-name-dialog';

import { useSessionsPaginated, useStopSession, useDeleteSession, useContinueSession, useUpdateSessionDisplayName, useRunnerTypes } from '@/services/queries';
import { useCurrentUser } from '@/services/queries/use-auth';
import { toast } from 'sonner';
import { useWorkspaceList } from '@/services/queries/use-workspace';
import { useProjectAccess } from '@/services/queries/use-project-access';
import { useDebounce } from '@/hooks/use-debounce';
import { useMemo } from 'react';
import { DEFAULT_PAGE_SIZE } from '@/types/api';

type ArtifactCountCellProps = {
  projectName: string;
  sessionName: string;
};

function ArtifactCountCell({ projectName, sessionName }: ArtifactCountCellProps) {
  const { data: files, isLoading } = useWorkspaceList(projectName, sessionName, 'artifacts');

  if (isLoading) {
    return <span className="text-sm text-muted-foreground/60">—</span>;
  }

  const fileCount = files ? files.filter((f) => !f.isDir).length : 0;

  if (fileCount === 0) {
    return <span className="text-sm text-muted-foreground/60">—</span>;
  }

  return (
    <div className="flex items-center gap-1 text-sm">
      <NotepadText className="h-3 w-3 text-muted-foreground" />
      <span>{fileCount}</span>
    </div>
  );
}

type SessionsSectionProps = {
  projectName: string;
};

export function SessionsSection({ projectName }: SessionsSectionProps) {
  // Pagination, search, and filter state
  const [searchInput, setSearchInput] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = DEFAULT_PAGE_SIZE;
  const [phaseFilter, setPhaseFilter] = useState<string>('');
  const [mySessionsOnly, setMySessionsOnly] = useState(false);
  const [sortBy, setSortBy] = useState<'created' | 'name'>('created');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');

  // Debounce search to avoid too many API calls
  const debouncedSearch = useDebounce(searchInput, 300);

  // Current user for "My sessions" filter
  const { data: currentUser, isLoading: isCurrentUserLoading } = useCurrentUser();

  // Reset offset when search or filters change
  useEffect(() => {
    setOffset(0);
  }, [debouncedSearch, phaseFilter, mySessionsOnly, sortBy, sortDirection]);

  // Access control (default-deny until role is resolved)
  const { data: access } = useProjectAccess(projectName);
  const canCreate = access?.userRole === 'edit' || access?.userRole === 'admin';
  const canDelete = access?.userRole === 'admin';
  const canModify = !!access?.userRole && access.userRole !== 'view';

  // Runner type lookup for display names
  const { data: runnerTypes } = useRunnerTypes(projectName);
  const runnerTypeMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const rt of runnerTypes ?? []) {
      map.set(rt.id, rt.displayName);
    }
    return map;
  }, [runnerTypes]);

  // React Query hooks with pagination
  const {
    data: paginatedData,
    isFetching,
    refetch,
  } = useSessionsPaginated(projectName, {
    limit,
    offset,
    search: debouncedSearch || undefined,
    phase: phaseFilter || undefined,
    userId: mySessionsOnly && currentUser?.userId ? currentUser.userId : undefined,
    sortBy,
    sortDirection,
  });

  const sessions = paginatedData?.items ?? [];
  const totalCount = paginatedData?.totalCount ?? 0;
  const hasMore = paginatedData?.hasMore ?? false;
  const currentPage = Math.floor(offset / limit) + 1;
  const totalPages = Math.ceil(totalCount / limit);

  const stopSessionMutation = useStopSession();
  const deleteSessionMutation = useDeleteSession();
  const continueSessionMutation = useContinueSession();
  const updateDisplayNameMutation = useUpdateSessionDisplayName();

  // State for edit name dialog
  const [editingSession, setEditingSession] = useState<{ name: string; displayName: string } | null>(null);

  const handleStop = async (sessionName: string) => {
    stopSessionMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => {
          toast.success(`Session "${sessionName}" stopped successfully`);
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to stop session');
        },
      }
    );
  };

  const handleDelete = async (sessionName: string) => {
    if (!confirm(`Delete agentic session "${sessionName}"? This action cannot be undone.`)) return;
    deleteSessionMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => {
          toast.success(`Session "${sessionName}" deleted successfully`);
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to delete session');
        },
      }
    );
  };

  const handleContinue = async (sessionName: string) => {
    continueSessionMutation.mutate(
      { projectName, parentSessionName: sessionName },
      {
        onSuccess: () => {
          toast.success(`Session "${sessionName}" restarted successfully`);
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to restart session');
        },
      }
    );
  };

  const handleNextPage = () => {
    if (hasMore) {
      setOffset(offset + limit);
    }
  };

  const handlePrevPage = () => {
    if (offset > 0) {
      setOffset(Math.max(0, offset - limit));
    }
  };

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchInput(e.target.value);
  };



  const handleEditName = (sessionName: string, currentDisplayName: string) => {
    setEditingSession({ name: sessionName, displayName: currentDisplayName });
  };

  const handleSaveEditName = (newName: string) => {
    if (!editingSession) return;

    updateDisplayNameMutation.mutate(
      {
        projectName,
        sessionName: editingSession.name,
        displayName: newName,
      },
      {
        onSuccess: () => {
          toast.success('Session name updated successfully');
          setEditingSession(null);
          refetch();
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to update session name');
        },
      }
    );
  };

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>Sessions</CardTitle>
            <CardDescription>
              Sessions scoped to this workspace
            </CardDescription>
          </div>
          <div className="flex gap-2">
            {canCreate && (
              <Button data-testid="new-session-btn" asChild>
                <Link href={`/projects/${projectName}/new`}>
                  <Plus className="w-4 h-4 mr-2" />
                  New Session
                </Link>
              </Button>
            )}
          </div>
        </div>
        {/* Search and filters */}
        <div className="flex items-center gap-3 mt-4 flex-wrap">
          <div className="relative max-w-sm flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search sessions..."
              value={searchInput}
              onChange={handleSearchChange}
              className="pl-9"
            />
          </div>

          <Select value={phaseFilter || 'all'} onValueChange={(value) => setPhaseFilter(value === 'all' ? '' : value)}>
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="Running,Pending,Creating">Active</SelectItem>
              <SelectItem value="Completed,Stopped">Completed</SelectItem>
              <SelectItem value="Failed">Failed</SelectItem>
            </SelectContent>
          </Select>

          <Button
            variant={mySessionsOnly ? 'default' : 'outline'}
            size="sm"
            onClick={() => setMySessionsOnly(!mySessionsOnly)}
            disabled={isCurrentUserLoading}
            className="h-9"
          >
            <User className="h-4 w-4 mr-1" />
            My sessions
          </Button>
        </div>

        {/* Active filter chips */}
        {(phaseFilter || mySessionsOnly) && (
          <div className="flex items-center gap-2 mt-2">
            <span className="text-xs text-muted-foreground">Filters:</span>
            {phaseFilter && (
              <Badge variant="secondary" className="cursor-pointer gap-1" onClick={() => setPhaseFilter('')}>
                {phaseFilter === 'Running,Pending,Creating' ? 'Active' : phaseFilter === 'Completed,Stopped' ? 'Completed' : phaseFilter}
                <span className="text-muted-foreground">&times;</span>
              </Badge>
            )}
            {mySessionsOnly && (
              <Badge variant="secondary" className="cursor-pointer gap-1" onClick={() => setMySessionsOnly(false)}>
                My sessions
                <span className="text-muted-foreground">&times;</span>
              </Badge>
            )}
          </div>
        )}
      </CardHeader>
      <CardContent>
        {(() => {
          const hasActiveFilters = !!debouncedSearch || !!phaseFilter || mySessionsOnly;
          if (sessions.length === 0 && !hasActiveFilters) {
            return (
              <EmptyState
                icon={Brain}
                title="No sessions found"
                description="Create your first agentic session"
              />
            );
          }
          if (sessions.length === 0 && hasActiveFilters) {
            return (
              <EmptyState
                icon={Search}
                title="No matching sessions"
                description="No sessions found matching the current filters"
              />
            );
          }
          return null;
        })()}
        {sessions.length > 0 && (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[20px] px-0"></TableHead>
                    <TableHead
                      className="min-w-[180px] cursor-pointer select-none"
                      tabIndex={0}
                      role="button"
                      aria-sort={sortBy === 'name' ? (sortDirection === 'asc' ? 'ascending' : 'descending') : 'none'}
                      onClick={() => {
                        if (sortBy === 'name') {
                          setSortDirection(prev => prev === 'asc' ? 'desc' : 'asc');
                        } else {
                          setSortBy('name');
                          setSortDirection('asc');
                        }
                      }}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          if (sortBy === 'name') {
                            setSortDirection(prev => prev === 'asc' ? 'desc' : 'asc');
                          } else {
                            setSortBy('name');
                            setSortDirection('asc');
                          }
                        }
                      }}
                    >
                      <div className="flex items-center gap-1">
                        Name
                        {sortBy === 'name' && (sortDirection === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                      </div>
                    </TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="hidden md:table-cell">Model</TableHead>
                    <TableHead
                      className="hidden lg:table-cell cursor-pointer select-none"
                      tabIndex={0}
                      role="button"
                      aria-sort={sortBy === 'created' ? (sortDirection === 'asc' ? 'ascending' : 'descending') : 'none'}
                      onClick={() => {
                        if (sortBy === 'created') {
                          setSortDirection(prev => prev === 'desc' ? 'asc' : 'desc');
                        } else {
                          setSortBy('created');
                          setSortDirection('desc');
                        }
                      }}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          if (sortBy === 'created') {
                            setSortDirection(prev => prev === 'desc' ? 'asc' : 'desc');
                          } else {
                            setSortBy('created');
                            setSortDirection('desc');
                          }
                        }
                      }}
                    >
                      <div className="flex items-center gap-1">
                        Created
                        {sortBy === 'created' && (sortDirection === 'desc' ? <ArrowDown className="h-3 w-3" /> : <ArrowUp className="h-3 w-3" />)}
                      </div>
                    </TableHead>
                    <TableHead className="hidden xl:table-cell">Creator</TableHead>
                    <TableHead className="hidden 2xl:table-cell">Artifacts</TableHead>
                    <TableHead className="w-[50px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sessions.map((session) => {
                    const sessionName = session.metadata.name;
                    const phase = session.status?.phase || 'Pending';
                    const isActionPending =
                      (stopSessionMutation.isPending && stopSessionMutation.variables?.sessionName === sessionName) ||
                      (deleteSessionMutation.isPending && deleteSessionMutation.variables?.sessionName === sessionName);

                    return (
                      <TableRow key={session.metadata?.uid || session.metadata?.name}>
                        <TableCell className="w-[20px] px-0 pr-1">
                          <SessionStatusDot phase={phase} />
                        </TableCell>
                        <TableCell className="font-medium min-w-[180px]">
                          <HoverCard openDelay={300} closeDelay={100}>
                            <HoverCardTrigger asChild>
                              <Link
                                href={`/projects/${projectName}/sessions/${session.metadata.name}`}
                                className="text-link hover:underline hover:text-link-hover transition-colors block"
                              >
                                <div>
                                  <div className="font-medium">{session.spec.displayName || session.metadata.name}</div>
                                  {session.spec.displayName && (
                                    <div className="text-xs text-muted-foreground font-normal">{session.metadata.name}</div>
                                  )}
                                </div>
                              </Link>
                            </HoverCardTrigger>
                            <HoverCardContent align="start" className="w-80">
                              <div className="space-y-2">
                                <div className="flex items-center justify-between">
                                  <p className="text-sm font-semibold truncate mr-2">
                                    {session.spec.displayName || session.metadata.name}
                                  </p>
                                  <AgentStatusIndicator
                                    status={session.status?.agentStatus ?? deriveAgentStatusFromPhase(phase)}
                                    compact
                                  />
                                </div>
                                {session.spec.displayName && (
                                  <p className="text-xs text-muted-foreground">{session.metadata.name}</p>
                                )}
                                <div className="flex flex-col gap-1.5 pt-1">
                                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                    <Cpu className="h-3 w-3" />
                                    <span>{session.spec.llmSettings.model}</span>
                                  </div>
                                  {session.metadata?.creationTimestamp && (
                                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                      <Clock className="h-3 w-3" />
                                      <span>{formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}</span>
                                    </div>
                                  )}
                                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                    <User className="h-3 w-3" />
                                    <span>{session.spec.userContext?.displayName || session.spec.userContext?.userId || '—'}</span>
                                  </div>
                                  {session.spec.initialPrompt && (
                                    <div className="flex items-start gap-1.5 text-xs text-muted-foreground pt-1">
                                      <MessageSquare className="h-3 w-3 mt-0.5 shrink-0" />
                                      <span className="line-clamp-3">{session.spec.initialPrompt}</span>
                                    </div>
                                  )}
                                </div>
                              </div>
                            </HoverCardContent>
                          </HoverCard>
                        </TableCell>
                        <TableCell>
                          <AgentStatusIndicator
                            status={session.status?.agentStatus ?? deriveAgentStatusFromPhase(phase)}
                          />
                        </TableCell>
                        <TableCell className="hidden md:table-cell">
                          <div className="text-sm truncate max-w-[160px]">
                            {(() => {
                              const runnerType = session.spec.environmentVariables?.RUNNER_TYPE;
                              const runnerLabel = runnerType ? runnerTypeMap.get(runnerType) : undefined;
                              return (
                                <>
                                  {runnerLabel && (
                                    <div className="text-xs font-medium text-foreground">{runnerLabel}</div>
                                  )}
                                  <div className="text-muted-foreground">{session.spec.llmSettings.model}</div>
                                </>
                              );
                            })()}
                          </div>
                        </TableCell>
                        <TableCell className="hidden lg:table-cell">
                          {session.metadata?.creationTimestamp &&
                            formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
                        </TableCell>
                        <TableCell className="hidden xl:table-cell">
                          <div className="text-sm text-muted-foreground truncate max-w-[140px]">
                            {session.spec.userContext?.displayName || session.spec.userContext?.userId || '—'}
                          </div>
                        </TableCell>
                        <TableCell className="hidden 2xl:table-cell">
                          <ArtifactCountCell projectName={projectName} sessionName={sessionName} />
                        </TableCell>
                        <TableCell>
                          {isActionPending ? (
                            <Button variant="ghost" size="sm" className="h-8 w-8 p-0" disabled>
                              <RefreshCw className="h-4 w-4 animate-spin" />
                            </Button>
                          ) : (
                            <SessionActions
                              sessionName={sessionName}
                              displayName={session.spec.displayName || sessionName}
                              phase={phase}
                              onStop={handleStop}
                              onContinue={handleContinue}
                              onDelete={handleDelete}
                              onEditName={handleEditName}
                              canDelete={canDelete}
                              canModify={canModify}
                            />
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>

            {/* Pagination controls */}
            {totalPages > 1 && (
              <div className="flex items-center justify-between pt-4 border-t mt-4">
                <div className="text-sm text-muted-foreground">
                  Showing {offset + 1}-{Math.min(offset + limit, totalCount)} of {totalCount} sessions
                </div>
                <Pagination className="mx-0 w-auto justify-end">
                  <PaginationContent>
                    <PaginationItem>
                      <PaginationPrevious
                        onClick={handlePrevPage}
                        className={offset === 0 || isFetching ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                      />
                    </PaginationItem>
                    {getPageNumbers(currentPage, totalPages).map((page, i) =>
                      page === 'ellipsis' ? (
                        <PaginationItem key={`ellipsis-${i}`}>
                          <PaginationEllipsis />
                        </PaginationItem>
                      ) : (
                        <PaginationItem key={page}>
                          <PaginationLink
                            isActive={page === currentPage}
                            onClick={() => { setOffset((page - 1) * limit); }}
                            className="cursor-pointer"
                          >
                            {page}
                          </PaginationLink>
                        </PaginationItem>
                      )
                    )}
                    <PaginationItem>
                      <PaginationNext
                        onClick={handleNextPage}
                        className={!hasMore || isFetching ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                      />
                    </PaginationItem>
                  </PaginationContent>
                </Pagination>
              </div>
            )}
          </>
        )}
      </CardContent>

      {/* Edit Session Name Dialog */}
      <EditSessionNameDialog
        open={!!editingSession}
        onOpenChange={(open) => !open && setEditingSession(null)}
        currentName={editingSession?.displayName || ''}
        onSave={handleSaveEditName}
        isLoading={updateDisplayNameMutation.isPending}
      />
    </Card>
  );
}

type SessionActionsProps = {
  sessionName: string;
  displayName: string;
  phase: string;
  onStop: (sessionName: string) => void;
  onContinue: (sessionName: string) => void;
  onDelete: (sessionName: string) => void;
  onEditName: (sessionName: string, currentDisplayName: string) => void;
  canDelete: boolean;
  canModify: boolean;
};

function SessionActions({ sessionName, displayName, phase, onStop, onContinue, onDelete, onEditName, canDelete, canModify }: SessionActionsProps) {
  type RowAction = {
    key: string;
    label: string;
    onClick: () => void;
    icon: React.ReactNode;
    className?: string;
  };

  const actions: RowAction[] = [];

  // Edit name is available for users who can modify
  if (canModify) {
    actions.push({
      key: 'edit',
      label: 'Edit name',
      onClick: () => onEditName(sessionName, displayName),
      icon: <Pencil className="h-4 w-4" />,
    });
  }

  // Stop is available for users who can modify
  if (canModify && (phase === 'Pending' || phase === 'Creating' || phase === 'Running')) {
    actions.push({
      key: 'stop',
      label: 'Stop',
      onClick: () => onStop(sessionName),
      icon: <Square className="h-4 w-4" />,
      className: 'text-orange-600',
    });
  }

  // Continue is available for users who can modify
  if (canModify && (phase === 'Completed' || phase === 'Failed' || phase === 'Stopped' || phase === 'Error')) {
    actions.push({
      key: 'continue',
      label: 'Continue',
      onClick: () => onContinue(sessionName),
      icon: <ArrowRight className="h-4 w-4" />,
      className: 'text-green-600',
    });
  }

  // Delete is only available for admins
  if (canDelete && phase !== 'Creating') {
    actions.push({
      key: 'delete',
      label: 'Delete',
      onClick: () => onDelete(sessionName),
      icon: <Trash2 className="h-4 w-4" />,
      className: 'text-red-600',
    });
  }

  // Only show dropdown if there are any actions available
  if (actions.length === 0) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
          <MoreVertical className="h-4 w-4" />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {actions.map((action) => (
          <DropdownMenuItem key={action.key} onClick={action.onClick} className={action.className}>
            {action.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
