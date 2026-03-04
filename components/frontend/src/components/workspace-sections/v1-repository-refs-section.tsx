'use client';

import { useState, useEffect } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { RefreshCw, Plus, Search, ChevronLeft, ChevronRight, GitFork, Loader2, Pencil } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { EmptyState } from '@/components/empty-state';

import {
  useV1RepositoryRefs,
  useV1CreateRepositoryRef,
  useV1UpdateRepositoryRef,
} from '@/services/queries/v1';
import { successToast, errorToast } from '@/hooks/use-toast';
import { useDebounce } from '@/hooks/use-debounce';
import type { RepositoryRef } from '@ambient-platform/sdk';

type V1RepositoryRefsSectionProps = {
  projectName: string;
};

export function V1RepositoryRefsSection({ projectName }: V1RepositoryRefsSectionProps) {
  const [searchInput, setSearchInput] = useState('');
  const [page, setPage] = useState(1);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [editingRef, setEditingRef] = useState<RepositoryRef | null>(null);
  const pageSize = 20;

  const debouncedSearch = useDebounce(searchInput, 300);

  const {
    data: listResponse,
    isFetching,
    refetch,
  } = useV1RepositoryRefs(projectName, {
    page,
    size: pageSize,
    search: debouncedSearch || undefined,
  });

  const refs = listResponse?.items ?? [];
  const total = listResponse?.total ?? 0;
  const totalPages = Math.ceil(total / pageSize);

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>Repositories (API Server)</CardTitle>
            <CardDescription>Manage repository references for this workspace</CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => refetch()} disabled={isFetching}>
              <RefreshCw className={`w-4 h-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <Button onClick={() => setShowAddDialog(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Add Repository
            </Button>
          </div>
        </div>
        <div className="relative mt-4 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search repositories..."
            value={searchInput}
            onChange={(e) => { setSearchInput(e.target.value); setPage(1); }}
            className="pl-9"
          />
        </div>
      </CardHeader>
      <CardContent>
        {refs.length === 0 && !debouncedSearch ? (
          <EmptyState
            icon={GitFork}
            title="No repositories configured"
            description="Add repository references to use in agentic sessions"
            action={{ label: 'Add Repository', onClick: () => setShowAddDialog(true) }}
          />
        ) : refs.length === 0 && debouncedSearch ? (
          <EmptyState
            icon={Search}
            title="No matching repositories"
            description={`No repositories found matching "${debouncedSearch}"`}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[150px]">Name</TableHead>
                    <TableHead>URL</TableHead>
                    <TableHead>Branch</TableHead>
                    <TableHead className="hidden md:table-cell">Provider</TableHead>
                    <TableHead className="hidden lg:table-cell">Created</TableHead>
                    <TableHead className="w-[50px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {refs.map((ref: RepositoryRef) => (
                    <TableRow key={ref.id}>
                      <TableCell className="font-medium">{ref.name}</TableCell>
                      <TableCell>
                        <span className="text-sm text-muted-foreground truncate max-w-[250px] block" title={ref.url}>
                          {ref.url}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{ref.branch || 'main'}</Badge>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <span className="text-sm capitalize">{ref.provider || '—'}</span>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {ref.created_at &&
                          formatDistanceToNow(new Date(ref.created_at), { addSuffix: true })}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0"
                          onClick={() => setEditingRef(ref)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
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

      <AddRepositoryRefDialog
        projectName={projectName}
        open={showAddDialog}
        onOpenChange={setShowAddDialog}
      />

      <EditRepositoryRefDialog
        projectName={projectName}
        repositoryRef={editingRef}
        open={!!editingRef}
        onOpenChange={(open) => { if (!open) setEditingRef(null); }}
      />
    </Card>
  );
}

function AddRepositoryRefDialog({
  projectName,
  open,
  onOpenChange,
}: {
  projectName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const createMutation = useV1CreateRepositoryRef(projectName);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [branch, setBranch] = useState('');

  const resetForm = () => {
    setName('');
    setUrl('');
    setBranch('');
  };

  const handleClose = () => {
    if (!createMutation.isPending) {
      resetForm();
      onOpenChange(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !url.trim()) return;

    createMutation.mutate(
      {
        name: name.trim(),
        url: url.trim(),
        ...(branch.trim() && { branch: branch.trim() }),
      },
      {
        onSuccess: () => {
          successToast(`Repository "${name}" added`);
          resetForm();
          onOpenChange(false);
        },
        onError: (err) => {
          errorToast(err instanceof Error ? err.message : 'Failed to add repository');
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Add Repository</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="repo-name">Name *</Label>
            <Input
              id="repo-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. my-frontend-repo"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="repo-url">URL *</Label>
            <Input
              id="repo-url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://github.com/org/repo"
            />
            <p className="text-xs text-muted-foreground">
              Provider and owner are auto-detected from the URL.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="repo-branch">Branch</Label>
            <Input
              id="repo-branch"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
              placeholder="main (default)"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={createMutation.isPending}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending || !name.trim() || !url.trim()}>
              {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Add Repository
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function EditRepositoryRefDialog({
  projectName,
  repositoryRef,
  open,
  onOpenChange,
}: {
  projectName: string;
  repositoryRef: RepositoryRef | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const updateMutation = useV1UpdateRepositoryRef(projectName);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [branch, setBranch] = useState('');

  useEffect(() => {
    if (open && repositoryRef) {
      setName(repositoryRef.name || '');
      setUrl(repositoryRef.url || '');
      setBranch(repositoryRef.branch || '');
    }
  }, [open, repositoryRef]);

  const handleClose = () => {
    if (!updateMutation.isPending) {
      onOpenChange(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!repositoryRef) return;

    updateMutation.mutate(
      {
        id: repositoryRef.id,
        data: {
          ...(name.trim() !== repositoryRef.name && { name: name.trim() }),
          ...(url.trim() !== repositoryRef.url && { url: url.trim() }),
          ...(branch.trim() !== (repositoryRef.branch || '') && { branch: branch.trim() }),
        },
      },
      {
        onSuccess: () => {
          successToast(`Repository "${name}" updated`);
          onOpenChange(false);
        },
        onError: (err) => {
          errorToast(err instanceof Error ? err.message : 'Failed to update repository');
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit Repository</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-repo-name">Name</Label>
            <Input
              id="edit-repo-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-repo-url">URL</Label>
            <Input
              id="edit-repo-url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-repo-branch">Branch</Label>
            <Input
              id="edit-repo-branch"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={updateMutation.isPending}>
              Cancel
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Save Changes
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
