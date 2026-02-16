'use client';

import { useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import {
  RefreshCw,
  Plus,
  Search,
  ChevronLeft,
  ChevronRight,
  KeyRound,
  Loader2,
  Trash2,
  Copy,
  Check,
  AlertTriangle,
} from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { EmptyState } from '@/components/empty-state';

import {
  useV1ProjectKeys,
  useV1CreateProjectKey,
  useV1DeleteProjectKey,
} from '@/services/queries/v1';
import { successToast, errorToast } from '@/hooks/use-toast';
import { useDebounce } from '@/hooks/use-debounce';
import type { ProjectKey } from '@ambient-platform/sdk';

type V1ProjectKeysSectionProps = {
  projectName: string;
};

export function V1ProjectKeysSection({ projectName }: V1ProjectKeysSectionProps) {
  const [searchInput, setSearchInput] = useState('');
  const [page, setPage] = useState(1);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [revokingKey, setRevokingKey] = useState<ProjectKey | null>(null);
  const pageSize = 20;

  const debouncedSearch = useDebounce(searchInput, 300);

  const {
    data: listResponse,
    isFetching,
    refetch,
  } = useV1ProjectKeys(projectName, {
    page,
    size: pageSize,
    search: debouncedSearch || undefined,
  });

  const keys = listResponse?.items ?? [];
  const total = listResponse?.total ?? 0;
  const totalPages = Math.ceil(total / pageSize);

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>API Keys (API Server)</CardTitle>
            <CardDescription>Manage API keys for programmatic access to this workspace</CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => refetch()} disabled={isFetching}>
              <RefreshCw className={`w-4 h-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <Button onClick={() => setShowCreateDialog(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Create Key
            </Button>
          </div>
        </div>
        <div className="relative mt-4 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search keys..."
            value={searchInput}
            onChange={(e) => { setSearchInput(e.target.value); setPage(1); }}
            className="pl-9"
          />
        </div>
      </CardHeader>
      <CardContent>
        {keys.length === 0 && !debouncedSearch ? (
          <EmptyState
            icon={KeyRound}
            title="No API keys"
            description="Create an API key for programmatic access to this workspace"
            action={{ label: 'Create Key', onClick: () => setShowCreateDialog(true) }}
          />
        ) : keys.length === 0 && debouncedSearch ? (
          <EmptyState
            icon={Search}
            title="No matching keys"
            description={`No API keys found matching "${debouncedSearch}"`}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[150px]">Name</TableHead>
                    <TableHead>Key Prefix</TableHead>
                    <TableHead className="hidden md:table-cell">Expires</TableHead>
                    <TableHead className="hidden lg:table-cell">Last Used</TableHead>
                    <TableHead className="hidden lg:table-cell">Created</TableHead>
                    <TableHead className="w-[50px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {keys.map((key: ProjectKey) => (
                    <TableRow key={key.id}>
                      <TableCell className="font-medium">{key.name}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="font-mono text-xs">
                          {key.key_prefix ? `${key.key_prefix}...` : '—'}
                        </Badge>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        {key.expires_at ? (
                          <span className="text-sm">
                            {formatDistanceToNow(new Date(key.expires_at), { addSuffix: true })}
                          </span>
                        ) : (
                          <span className="text-sm text-muted-foreground">Never</span>
                        )}
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {key.last_used_at ? (
                          <span className="text-sm">
                            {formatDistanceToNow(new Date(key.last_used_at), { addSuffix: true })}
                          </span>
                        ) : (
                          <span className="text-sm text-muted-foreground">Never</span>
                        )}
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {key.created_at &&
                          formatDistanceToNow(new Date(key.created_at), { addSuffix: true })}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                          onClick={() => setRevokingKey(key)}
                        >
                          <Trash2 className="h-4 w-4" />
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

      <CreateProjectKeyDialog
        projectName={projectName}
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
      />

      <RevokeProjectKeyDialog
        projectName={projectName}
        projectKey={revokingKey}
        open={!!revokingKey}
        onOpenChange={(open) => { if (!open) setRevokingKey(null); }}
      />
    </Card>
  );
}

function CreateProjectKeyDialog({
  projectName,
  open,
  onOpenChange,
}: {
  projectName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const createMutation = useV1CreateProjectKey(projectName);
  const [name, setName] = useState('');
  const [createdKey, setCreatedKey] = useState<ProjectKey | null>(null);
  const [copied, setCopied] = useState(false);

  const resetForm = () => {
    setName('');
    setCreatedKey(null);
    setCopied(false);
  };

  const handleClose = () => {
    if (!createMutation.isPending) {
      resetForm();
      onOpenChange(false);
    }
  };

  const handleCopy = async () => {
    if (!createdKey?.plaintext_key) return;
    try {
      await navigator.clipboard.writeText(createdKey.plaintext_key);
      setCopied(true);
      successToast('API key copied to clipboard');
      setTimeout(() => setCopied(false), 2000);
    } catch {
      errorToast('Failed to copy to clipboard');
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    createMutation.mutate(
      { name: name.trim() },
      {
        onSuccess: (key) => {
          setCreatedKey(key);
          successToast(`API key "${name}" created`);
        },
        onError: (err) => {
          errorToast(err instanceof Error ? err.message : 'Failed to create API key');
        },
      }
    );
  };

  if (createdKey) {
    return (
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>API Key Created</DialogTitle>
            <DialogDescription>
              Copy your API key now. You will not be able to see it again.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="p-4 bg-amber-50 border border-amber-200 rounded-md dark:bg-amber-950/50 dark:border-amber-800">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
                <p className="text-sm text-amber-800 dark:text-amber-200">
                  This is the only time the full key will be displayed. Store it securely.
                </p>
              </div>
            </div>
            <div className="space-y-2">
              <Label>Key Name</Label>
              <p className="text-sm font-medium">{createdKey.name}</p>
            </div>
            <div className="space-y-2">
              <Label>API Key</Label>
              <div className="flex gap-2">
                <Input
                  readOnly
                  value={createdKey.plaintext_key}
                  className="font-mono text-xs"
                />
                <Button
                  variant="outline"
                  size="sm"
                  className="shrink-0"
                  onClick={handleCopy}
                >
                  {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </Button>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={handleClose}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Create API Key</DialogTitle>
          <DialogDescription>
            Create an API key for programmatic access to this workspace.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="key-name">Key Name *</Label>
            <Input
              id="key-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. ci-deploy-key"
            />
            <p className="text-xs text-muted-foreground">
              A descriptive name to identify this key.
            </p>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={createMutation.isPending}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending || !name.trim()}>
              {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Create Key
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function RevokeProjectKeyDialog({
  projectName,
  projectKey,
  open,
  onOpenChange,
}: {
  projectName: string;
  projectKey: ProjectKey | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const deleteMutation = useV1DeleteProjectKey(projectName);

  const handleRevoke = () => {
    if (!projectKey) return;

    deleteMutation.mutate(projectKey.id, {
      onSuccess: () => {
        successToast(`API key "${projectKey.name}" revoked`);
        onOpenChange(false);
      },
      onError: (err) => {
        errorToast(err instanceof Error ? err.message : 'Failed to revoke API key');
      },
    });
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke API Key</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to revoke the API key &quot;{projectKey?.name}&quot;?
            This action cannot be undone. Any applications using this key will lose access immediately.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleteMutation.isPending}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleRevoke}
            disabled={deleteMutation.isPending}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {deleteMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Revoke Key
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
