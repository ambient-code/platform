'use client';

import { useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { RefreshCw, Plus, Search, ChevronLeft, ChevronRight, Shield, Loader2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { EmptyState } from '@/components/empty-state';

import { useV1Permissions, useV1CreatePermission } from '@/services/queries/v1';
import { successToast, errorToast } from '@/hooks/use-toast';
import { useDebounce } from '@/hooks/use-debounce';
import type { Permission } from '@ambient-platform/sdk';

type V1PermissionsSectionProps = {
  projectName: string;
};

const ROLE_COLORS: Record<string, string> = {
  admin: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
  edit: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  view: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
};

export function V1PermissionsSection({ projectName }: V1PermissionsSectionProps) {
  const [searchInput, setSearchInput] = useState('');
  const [page, setPage] = useState(1);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const pageSize = 20;

  const debouncedSearch = useDebounce(searchInput, 300);

  const {
    data: listResponse,
    isFetching,
    refetch,
  } = useV1Permissions(projectName, {
    page,
    size: pageSize,
    search: debouncedSearch || undefined,
  });

  const permissions = listResponse?.items ?? [];
  const total = listResponse?.total ?? 0;
  const totalPages = Math.ceil(total / pageSize);

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>Permissions (API Server)</CardTitle>
            <CardDescription>Manage access permissions for this workspace</CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => refetch()} disabled={isFetching}>
              <RefreshCw className={`w-4 h-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <Button onClick={() => setShowAddDialog(true)}>
              <Plus className="w-4 h-4 mr-2" />
              Add Permission
            </Button>
          </div>
        </div>
        <div className="relative mt-4 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search permissions..."
            value={searchInput}
            onChange={(e) => { setSearchInput(e.target.value); setPage(1); }}
            className="pl-9"
          />
        </div>
      </CardHeader>
      <CardContent>
        {permissions.length === 0 && !debouncedSearch ? (
          <EmptyState
            icon={Shield}
            title="No permissions configured"
            description="Add permissions to control access to this workspace"
            action={{ label: 'Add Permission', onClick: () => setShowAddDialog(true) }}
          />
        ) : permissions.length === 0 && debouncedSearch ? (
          <EmptyState
            icon={Search}
            title="No matching permissions"
            description={`No permissions found matching "${debouncedSearch}"`}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Subject</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead className="hidden lg:table-cell">Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {permissions.map((perm: Permission) => (
                    <TableRow key={perm.id}>
                      <TableCell className="font-medium">{perm.subject_name}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="capitalize">{perm.subject_type}</Badge>
                      </TableCell>
                      <TableCell>
                        <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${ROLE_COLORS[perm.role] || ''}`}>
                          {perm.role}
                        </span>
                      </TableCell>
                      <TableCell className="hidden lg:table-cell">
                        {perm.created_at &&
                          formatDistanceToNow(new Date(perm.created_at), { addSuffix: true })}
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

      <AddPermissionDialog
        projectName={projectName}
        open={showAddDialog}
        onOpenChange={setShowAddDialog}
      />
    </Card>
  );
}

function AddPermissionDialog({
  projectName,
  open,
  onOpenChange,
}: {
  projectName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const createMutation = useV1CreatePermission(projectName);
  const [subjectName, setSubjectName] = useState('');
  const [subjectType, setSubjectType] = useState<string>('user');
  const [role, setRole] = useState<string>('view');

  const resetForm = () => {
    setSubjectName('');
    setSubjectType('user');
    setRole('view');
  };

  const handleClose = () => {
    if (!createMutation.isPending) {
      resetForm();
      onOpenChange(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!subjectName.trim()) return;

    createMutation.mutate(
      {
        subject_name: subjectName.trim(),
        subject_type: subjectType,
        role,
      },
      {
        onSuccess: () => {
          successToast(`Permission added for ${subjectName}`);
          resetForm();
          onOpenChange(false);
        },
        onError: (err) => {
          errorToast(err instanceof Error ? err.message : 'Failed to add permission');
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Add Permission</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="perm-subject-name">Subject Name *</Label>
            <Input
              id="perm-subject-name"
              value={subjectName}
              onChange={(e) => setSubjectName(e.target.value)}
              placeholder="e.g. alice or engineering-team"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="perm-subject-type">Subject Type</Label>
            <Select value={subjectType} onValueChange={setSubjectType}>
              <SelectTrigger id="perm-subject-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="group">Group</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="perm-role">Role</Label>
            <Select value={role} onValueChange={setRole}>
              <SelectTrigger id="perm-role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="edit">Edit</SelectItem>
                <SelectItem value="view">View</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={createMutation.isPending}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending || !subjectName.trim()}>
              {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Add Permission
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
