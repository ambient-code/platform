'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import { Plus, RefreshCw, Trash2, FolderOpen, Loader2, Search, ChevronLeft, ChevronRight } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Input } from '@/components/ui/input';
import { useProjectsPaginated, useDeleteProject } from '@/services/queries';
import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/empty-state';
import { ErrorMessage } from '@/components/error-message';
import { DestructiveConfirmationDialog } from '@/components/confirmation-dialog';
import { CreateWorkspaceDialog } from '@/components/create-workspace-dialog';
import { successToast, errorToast } from '@/hooks/use-toast';
import type { Project } from '@/types/api';
import { DEFAULT_PAGE_SIZE } from '@/types/api';
import { useDebounce } from '@/hooks/use-debounce';

export default function ProjectsPage() {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [projectToDelete, setProjectToDelete] = useState<Project | null>(null);
  const [showCreateDialog, setShowCreateDialog] = useState(false);

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
    isLoading,
    isFetching,
    error,
    refetch,
  } = useProjectsPaginated({
    limit,
    offset,
    search: debouncedSearch || undefined,
  });

  const projects = paginatedData?.items ?? [];
  const totalCount = paginatedData?.totalCount ?? 0;
  const hasMore = paginatedData?.hasMore ?? false;
  const currentPage = Math.floor(offset / limit) + 1;
  const totalPages = Math.ceil(totalCount / limit);

  const deleteProjectMutation = useDeleteProject();

  const handleRefreshClick = () => {
    refetch();
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

  const openDeleteDialog = (project: Project) => {
    setProjectToDelete(project);
    setShowDeleteDialog(true);
  };

  const closeDeleteDialog = () => {
    setShowDeleteDialog(false);
    setProjectToDelete(null);
  };

  const confirmDelete = async () => {
    if (!projectToDelete) return;

    deleteProjectMutation.mutate(projectToDelete.name, {
      onSuccess: () => {
        successToast(`Project "${projectToDelete.displayName || projectToDelete.name}" deleted successfully`);
        closeDeleteDialog();
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to delete project');
      },
    });
  };

  // Initial loading state (no data yet)
  if (isLoading && !paginatedData) {
    return (
      <div className="min-h-screen bg-background">
        <div className="container mx-auto p-6">
          <div className="flex items-center justify-center h-64">
            <Alert className="max-w-md mx-4">
              <Loader2 className="h-4 w-4 animate-spin" />
              <AlertTitle>Loading Workspaces...</AlertTitle>
              <AlertDescription>
                <p>Gathering information on existing workspaces.</p>
              </AlertDescription>
            </Alert>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 border-b">
        <div className="container mx-auto px-6 py-4">
          <PageHeader
            title="Workspaces"
            description="Workspaces are isolated environments for organizing AI-powered agentic sessions. Create a workspace to get started."
          />
        </div>
      </div>

      <div className="container mx-auto p-0">
        {/* Error state */}
        {error && (
          <div className="px-6 pt-4">
            <ErrorMessage error={error} onRetry={() => refetch()} />
          </div>
        )}

        {/* Content */}
        <div className="px-6 pt-4">
        <Card>
          <CardHeader>
            <div className="flex items-start justify-between">
              <div>
                <CardTitle>Workspaces</CardTitle>
                <CardDescription>
                  Each workspace provides an isolated environment for your team to run agentic sessions, manage API keys, and control access permissions
                </CardDescription>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={handleRefreshClick}
                  disabled={isFetching}
                >
                  <RefreshCw
                    className={`w-4 h-4 mr-2 ${isFetching ? 'animate-spin' : ''}`}
                  />
                  Refresh
                </Button>
                <Button onClick={() => setShowCreateDialog(true)}>
                  <Plus className="w-4 h-4 mr-2" />
                  New Workspace
                </Button>
              </div>
            </div>
            {/* Search input */}
            <div className="relative mt-4 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search workspaces..."
                value={searchInput}
                onChange={handleSearchChange}
                className="pl-9"
              />
            </div>
          </CardHeader>
          <CardContent>
            {projects.length === 0 && !debouncedSearch ? (
              <EmptyState
                icon={FolderOpen}
                title="No projects found"
                description="Get started by creating your first project"
                action={{
                  label: 'Create Workspace',
                  onClick: () => setShowCreateDialog(true),
                }}
              />
            ) : projects.length === 0 && debouncedSearch ? (
              <EmptyState
                icon={Search}
                title="No matching workspaces"
                description={`No workspaces found matching "${debouncedSearch}"`}
              />
            ) : (
              <>
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="min-w-[200px]">Name</TableHead>
                        <TableHead className="hidden md:table-cell">
                          Description
                        </TableHead>
                        <TableHead className="hidden lg:table-cell">
                          Created
                        </TableHead>
                        <TableHead className="w-[50px]">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {projects.map((project) => (
                        <TableRow key={project.name}>
                          <TableCell className="font-medium min-w-[200px]">
                            <Link
                              href={`/projects/${project.name}`}
                              className="text-link hover:underline hover:text-link-hover transition-colors block"
                            >
                              <div>
                                <div className="font-medium">
                                  {project.displayName || project.name}
                                </div>
                                <div className="text-xs text-muted-foreground font-normal">
                                  {project.name}
                                </div>
                              </div>
                            </Link>
                          </TableCell>
                          <TableCell className="hidden md:table-cell max-w-[200px]">
                            <span
                              className="truncate block"
                              title={project.description || '—'}
                            >
                              {project.description || '—'}
                            </span>
                          </TableCell>
                          <TableCell className="hidden lg:table-cell">
                            {project.creationTimestamp ? (
                              <span>
                                {formatDistanceToNow(
                                  new Date(project.creationTimestamp),
                                  { addSuffix: true }
                                )}
                              </span>
                            ) : (
                              <span>—</span>
                            )}
                          </TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-8 w-8 p-0"
                              onClick={() => openDeleteDialog(project)}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                {/* Pagination controls */}
                {totalPages > 1 && (
                  <div className="flex items-center justify-between pt-4 border-t mt-4">
                    <div className="text-sm text-muted-foreground">
                      Showing {offset + 1}-{Math.min(offset + limit, totalCount)} of {totalCount} workspaces
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
        </Card>
      </div>

        {/* Delete confirmation dialog */}
        <DestructiveConfirmationDialog
          open={showDeleteDialog}
          onOpenChange={setShowDeleteDialog}
          onConfirm={confirmDelete}
          title="Delete workspace"
          description={`Are you sure you want to delete workspace "${projectToDelete?.name}"? This will permanently remove the workspace and all related resources. This action cannot be undone.`}
          confirmText="Delete"
          loading={deleteProjectMutation.isPending}
        />

        {/* Create workspace dialog */}
        <CreateWorkspaceDialog
          open={showCreateDialog}
          onOpenChange={setShowCreateDialog}
        />
      </div>
    </div>
  );
}
