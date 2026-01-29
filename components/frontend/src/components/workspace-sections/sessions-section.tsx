'use client';

import { useState, useEffect } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { Plus, RefreshCw, MoreVertical, Square, Trash2, ArrowRight, Brain, Search, ChevronLeft, ChevronRight, Pencil } from 'lucide-react';
import Link from 'next/link';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { EmptyState } from '@/components/empty-state';
import { SessionPhaseBadge } from '@/components/status-badge';
import { CreateSessionDialog } from '@/components/create-session-dialog';
import { EditSessionNameDialog } from '@/components/edit-session-name-dialog';

import { useSessionsPaginated, useStopSession, useDeleteSession, useContinueSession, useUpdateSessionDisplayName } from '@/services/queries';
import { successToast, errorToast } from '@/hooks/use-toast';
import { useDebounce } from '@/hooks/use-debounce';
import { DEFAULT_PAGE_SIZE } from '@/types/api';

type SessionsSectionProps = {
  projectName: string;
};

export function SessionsSection({ projectName }: SessionsSectionProps) {
  // Pagination and search state
  const [searchInput, setSearchInput] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = DEFAULT_PAGE_SIZE;

  // Debounce search to avoid too many API calls
  const debouncedSearch = useDebounce(searchInput, 300);

  // Reset offset when search changes
  useEffect(() => {
    setOffset(0);
  }, [debouncedSearch]);

  // React Query hooks with pagination
  const {
    data: paginatedData,
    isFetching,
    refetch,
  } = useSessionsPaginated(projectName, {
    limit,
    offset,
    search: debouncedSearch || undefined,
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
          successToast(`Session "${sessionName}" stopped successfully`);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to stop session');
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
          successToast(`Session "${sessionName}" deleted successfully`);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to delete session');
        },
      }
    );
  };

  const handleContinue = async (sessionName: string) => {
    continueSessionMutation.mutate(
      { projectName, parentSessionName: sessionName },
      {
        onSuccess: () => {
          successToast(`Session "${sessionName}" restarted successfully`);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to restart session');
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

  const handleRefresh = () => {
    setOffset(0);
    refetch();
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
          successToast('Session name updated successfully');
          setEditingSession(null);
          refetch();
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to update session name');
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
            <Button variant="outline" onClick={handleRefresh} disabled={isFetching}>
              <RefreshCw className={`w-4 h-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <CreateSessionDialog
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
        {/* Search input */}
        <div className="relative mt-4 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search sessions..."
            value={searchInput}
            onChange={handleSearchChange}
            className="pl-9"
          />
        </div>
      </CardHeader>
      <CardContent>
        {sessions.length === 0 && !debouncedSearch ? (
          <EmptyState
            icon={Brain}
            title="No sessions found"
            description="Create your first agentic session"
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
                    <TableHead>Status</TableHead>
                    <TableHead>Mode</TableHead>
                    <TableHead className="hidden md:table-cell">Model</TableHead>
                    <TableHead className="hidden lg:table-cell">Created</TableHead>
                    <TableHead className="hidden xl:table-cell">Cost</TableHead>
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
                        <TableCell className="font-medium min-w-[180px]">
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
                        </TableCell>
                        <TableCell>
                          <SessionPhaseBadge phase={phase} />
                        </TableCell>
                        <TableCell>
                          <span className="text-xs px-2 py-1 rounded border bg-muted/50">
                            {session.spec?.interactive ? 'Interactive' : 'Headless'}
                          </span>
                        </TableCell>
                        <TableCell className="hidden md:table-cell">
                          <span className="text-sm text-muted-foreground truncate max-w-[120px] block">
                            {session.spec.llmSettings.model}
                          </span>
                        </TableCell>
                        <TableCell className="hidden lg:table-cell">
                          {session.metadata?.creationTimestamp &&
                            formatDistanceToNow(new Date(session.metadata.creationTimestamp), { addSuffix: true })}
                        </TableCell>
                        <TableCell className="hidden xl:table-cell">
                          {/* total_cost_usd removed from simplified status */}
                          <span className="text-sm text-muted-foreground/60">—</span>
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
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handlePrevPage}
                    disabled={offset === 0 || isFetching}
                  >
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    Previous
                  </Button>
                  <span className="text-sm text-muted-foreground px-2">
                    Page {currentPage} of {totalPages}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleNextPage}
                    disabled={!hasMore || isFetching}
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
};

function SessionActions({ sessionName, displayName, phase, onStop, onContinue, onDelete, onEditName }: SessionActionsProps) {
  type RowAction = {
    key: string;
    label: string;
    onClick: () => void;
    icon: React.ReactNode;
    className?: string;
  };

  const actions: RowAction[] = [];

  // Edit name is always available
  actions.push({
    key: 'edit',
    label: 'Edit name',
    onClick: () => onEditName(sessionName, displayName),
    icon: <Pencil className="h-4 w-4" />,
  });

  if (phase === 'Pending' || phase === 'Creating' || phase === 'Running') {
    actions.push({
      key: 'stop',
      label: 'Stop',
      onClick: () => onStop(sessionName),
      icon: <Square className="h-4 w-4" />,
      className: 'text-orange-600',
    });
  }

  if (phase === 'Completed' || phase === 'Failed' || phase === 'Stopped' || phase === 'Error') {
    actions.push({
      key: 'continue',
      label: 'Continue',
      onClick: () => onContinue(sessionName),
      icon: <ArrowRight className="h-4 w-4" />,
      className: 'text-green-600',
    });
  }

  if (phase !== 'Creating') {
    actions.push({
      key: 'delete',
      label: 'Delete',
      onClick: () => onDelete(sessionName),
      icon: <Trash2 className="h-4 w-4" />,
      className: 'text-red-600',
    });
  }

  // Always show dropdown since we now always have at least the edit action
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
