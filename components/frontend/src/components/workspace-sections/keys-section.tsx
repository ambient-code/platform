'use client';

import { useCallback, useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { Copy, KeyRound, Loader2, Plus, RefreshCw, Trash2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { ErrorMessage } from '@/components/error-message';
import { EmptyState } from '@/components/empty-state';
import { DestructiveConfirmationDialog } from '@/components/confirmation-dialog';

import { useKeys, useCreateKey, useDeleteKey } from '@/services/queries';
import { successToast, errorToast } from '@/hooks/use-toast';
import type { CreateKeyRequest } from '@/services/api/keys';
import { ROLE_DEFINITIONS } from '@/lib/role-colors';

type KeysSectionProps = {
  projectName: string;
};

const EXPIRATION_OPTIONS = [
  { value: '86400', label: '1 day' },
  { value: '604800', label: '7 days' },
  { value: '2592000', label: '30 days' },
  { value: '7776000', label: '90 days' },
  { value: '31536000', label: '1 year' },
  { value: 'none', label: 'No expiration' },
] as const;

export function KeysSection({ projectName }: KeysSectionProps) {
  const { data: keys = [], isLoading, error, refetch } = useKeys(projectName);
  const createKeyMutation = useCreateKey();
  const deleteKeyMutation = useDeleteKey();

  const [showCreate, setShowCreate] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyDesc, setNewKeyDesc] = useState('');
  const [newKeyRole, setNewKeyRole] = useState<'view' | 'edit' | 'admin'>('edit');
  const [newKeyExpiration, setNewKeyExpiration] = useState('7776000'); // default 90 days
  const [oneTimeKey, setOneTimeKey] = useState<string | null>(null);
  const [oneTimeKeyName, setOneTimeKeyName] = useState<string>('');
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [keyToDelete, setKeyToDelete] = useState<{ id: string; name: string } | null>(null);

  const handleCreate = useCallback(() => {
    if (!newKeyName.trim()) return;

    const request: CreateKeyRequest = {
      name: newKeyName.trim(),
      description: newKeyDesc.trim() || undefined,
      role: newKeyRole,
      expirationSeconds: newKeyExpiration !== 'none' ? Number(newKeyExpiration) : undefined,
    };

    createKeyMutation.mutate(
      { projectName, data: request },
      {
        onSuccess: (data) => {
          successToast(`Access key "${data.name}" created successfully`);
          setOneTimeKey(data.key);
          setOneTimeKeyName(data.name);
          setNewKeyName('');
          setNewKeyDesc('');
          setNewKeyExpiration('7776000');
          setShowCreate(false);
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to create key');
        },
      }
    );
  }, [newKeyName, newKeyDesc, newKeyRole, newKeyExpiration, projectName, createKeyMutation]);

  const openDeleteDialog = useCallback((keyId: string, keyName: string) => {
    setKeyToDelete({ id: keyId, name: keyName });
    setShowDeleteDialog(true);
  }, []);

  const confirmDelete = useCallback(() => {
    if (!keyToDelete) return;
    deleteKeyMutation.mutate(
      { projectName, keyId: keyToDelete.id },
      {
        onSuccess: () => {
          successToast(`Access key "${keyToDelete.name}" deleted successfully`);
          setShowDeleteDialog(false);
          setKeyToDelete(null);
        },
      }
    );
  }, [keyToDelete, projectName, deleteKeyMutation]);

  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
    } catch {}
  };

  if (isLoading && keys.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="animate-spin h-8 w-8" />
        <span className="ml-2">Loading access keys...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold flex items-center gap-2">
            <KeyRound className="w-5 h-5" />
            Access Keys
          </h2>
          <p className="text-sm text-muted-foreground mt-1">
            Create and manage API keys for non-user access
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="w-4 h-4 mr-2" />
            Create Key
          </Button>
          <Button variant="outline" onClick={() => refetch()} disabled={isLoading}>
            <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>
      </div>

      {/* Error state */}
      {error && <ErrorMessage error={error} onRetry={() => refetch()} />}
      {createKeyMutation.isError && (
        <ErrorMessage error={createKeyMutation.error} />
      )}
      {deleteKeyMutation.isError && (
        <ErrorMessage error={deleteKeyMutation.error} />
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyRound className="w-5 h-5" />
            Access Keys ({keys.length})
          </CardTitle>
          <CardDescription>API keys scoped to this workspace</CardDescription>
        </CardHeader>
        <CardContent>
          {keys.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {keys.map((k) => {
                  const isDeletingThis = deleteKeyMutation.isPending && deleteKeyMutation.variables?.keyId === k.id;
                  return (
                    <TableRow key={k.id}>
                      <TableCell className="font-medium">{k.name}</TableCell>
                      <TableCell>
                        {k.description || (
                          <span className="text-muted-foreground italic">No description</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {k.createdAt ? (
                          formatDistanceToNow(new Date(k.createdAt), { addSuffix: true })
                        ) : (
                          <span className="text-muted-foreground">Unknown</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {k.lastUsedAt ? (
                          formatDistanceToNow(new Date(k.lastUsedAt), { addSuffix: true })
                        ) : (
                          <span className="text-muted-foreground">Never</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {k.role ? (
                          (() => {
                            const role = k.role as keyof typeof ROLE_DEFINITIONS;
                            const cfg = ROLE_DEFINITIONS[role];
                            const Icon = cfg.icon;
                            return (
                              <Badge className={cfg.color} style={{ cursor: 'default' }}>
                                <Icon className="w-3 h-3 mr-1" />
                                {cfg.label}
                              </Badge>
                            );
                          })()
                        ) : (
                          <span className="text-muted-foreground">&mdash;</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => openDeleteDialog(k.id, k.name)}
                          disabled={isDeletingThis}
                        >
                          {isDeletingThis ? (
                            <Loader2 className="w-4 h-4 animate-spin" />
                          ) : (
                            <Trash2 className="w-4 h-4" />
                          )}
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          ) : (
            <EmptyState
              icon={KeyRound}
              title="No access keys"
              description="Create an API key to enable non-user access"
              action={{
                label: 'Create Your First Key',
                onClick: () => setShowCreate(true),
              }}
            />
          )}
        </CardContent>
      </Card>

      {/* Create Key Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Create Access Key</DialogTitle>
            <DialogDescription>Provide a name and configure the key</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="key-name">Name *</Label>
              <Input
                id="key-name"
                value={newKeyName}
                onChange={(e) => setNewKeyName(e.target.value)}
                placeholder="my-ci-key"
                maxLength={64}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="key-desc">Description</Label>
              <Input
                id="key-desc"
                value={newKeyDesc}
                onChange={(e) => setNewKeyDesc(e.target.value)}
                placeholder="Used by CI pipelines"
                maxLength={200}
              />
            </div>
            <div className="space-y-2">
              <Label>Role</Label>
              <div className="space-y-3">
                {(['view', 'edit', 'admin'] as const).map((roleKey) => {
                  const cfg = ROLE_DEFINITIONS[roleKey];
                  const Icon = cfg.icon;
                  const id = `key-role-${roleKey}`;
                  return (
                    <div key={roleKey} className="flex items-start gap-3">
                      <input
                        type="radio"
                        name="key-role"
                        id={id}
                        className="mt-1 h-4 w-4"
                        value={roleKey}
                        checked={newKeyRole === roleKey}
                        onChange={() => setNewKeyRole(roleKey)}
                        disabled={createKeyMutation.isPending}
                      />
                      <Label htmlFor={id} className="flex-1 cursor-pointer">
                        <div className="flex items-center gap-2">
                          <Icon className="w-4 h-4" />
                          <span className="font-medium">{cfg.label}</span>
                        </div>
                        <div className="text-sm text-muted-foreground ml-6">{cfg.description}</div>
                      </Label>
                    </div>
                  );
                })}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="key-expiration">Token Lifetime</Label>
              <Select value={newKeyExpiration} onValueChange={setNewKeyExpiration}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Select lifetime" />
                </SelectTrigger>
                <SelectContent>
                  {EXPIRATION_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                How long the token remains valid. Choose &quot;No expiration&quot; for long-lived service keys.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowCreate(false)}
              disabled={createKeyMutation.isPending}
            >
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={createKeyMutation.isPending || !newKeyName.trim()}>
              {createKeyMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Creating...
                </>
              ) : (
                'Create Key'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* One-time Key Viewer */}
      <Dialog open={oneTimeKey !== null} onOpenChange={(open) => !open && setOneTimeKey(null)}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Copy Your New Access Key</DialogTitle>
            <DialogDescription>
              This is the only time the full key will be shown. Store it securely. Key name: <b>{oneTimeKeyName}</b>
            </DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2">
            <code className="text-sm bg-muted px-2 py-2 rounded break-all w-full">{oneTimeKey || ''}</code>
            <Button variant="ghost" size="sm" onClick={() => oneTimeKey && copy(oneTimeKey)}>
              <Copy className="w-4 h-4" />
            </Button>
          </div>
          <DialogFooter>
            <Button onClick={() => setOneTimeKey(null)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation dialog */}
      <DestructiveConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        onConfirm={confirmDelete}
        title="Delete Access Key"
        description={`Are you sure you want to delete the access key "${keyToDelete?.name}"? This action cannot be undone and any systems using this key will lose access.`}
        confirmText="Delete Key"
        loading={deleteKeyMutation.isPending}
      />
    </div>
  );
}
