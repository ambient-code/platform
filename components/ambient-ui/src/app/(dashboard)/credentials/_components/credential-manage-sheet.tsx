'use client'

import { useState } from 'react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import type { DomainCredential } from '@/domain/types'
import { getProviderMeta, getCategoryForProvider } from '@/domain/credential-providers'
import { formatRelativeTime, formatAbsoluteTime } from '@/lib/format-timestamp'
import { useUpdateCredential, useDeleteCredential } from '@/queries/use-credentials'
import { useRoleBindings } from '@/queries/use-role-bindings'

export function CredentialManageSheet({
  credential,
  open,
  onOpenChange,
}: {
  credential: DomainCredential | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [newToken, setNewToken] = useState('')
  const [rotateError, setRotateError] = useState<string | null>(null)
  const updateCredential = useUpdateCredential()
  const deleteCredential = useDeleteCredential()

  const { data: bindingsData } = useRoleBindings(
    credential
      ? { search: `credential_id = '${credential.id}'` }
      : undefined,
  )

  if (!credential) return null

  const providerMeta = getProviderMeta(credential.provider)
  const category = getCategoryForProvider(credential.provider)
  const bindingCount = bindingsData?.items.length ?? 0

  async function handleRotateToken() {
    if (!credential || !newToken) return
    setRotateError(null)

    try {
      await updateCredential.mutateAsync({
        id: credential.id,
        request: { token: newToken },
      })
      setNewToken('')
    } catch (err) {
      console.error('rotate token failed', err)
      setRotateError('Failed to rotate token. Please try again.')
    }
  }

  async function handleDelete() {
    if (!credential) return
    try {
      await deleteCredential.mutateAsync(credential.id)
      onOpenChange(false)
    } catch (err) {
      console.error('delete credential failed', err)
    }
  }

  function handleClose(v: boolean) {
    if (!v) {
      setNewToken('')
      setRotateError(null)
    }
    onOpenChange(v)
  }

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent side="right" className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{credential.name}</SheetTitle>
          <SheetDescription>
            Manage credential settings and access.
          </SheetDescription>
        </SheetHeader>

        <div className="flex flex-col gap-6 px-4 pb-4">
          {/* Metadata */}
          <div className="space-y-3">
            <h3 className="text-sm font-medium text-muted-foreground">Details</h3>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <span className="text-muted-foreground">Provider</span>
              <div>
                <Badge variant="outline">
                  {providerMeta?.label ?? credential.provider}
                </Badge>
              </div>

              <span className="text-muted-foreground">Category</span>
              <span>{category ?? 'Unknown'}</span>

              <span className="text-muted-foreground">Created</span>
              <span title={formatAbsoluteTime(credential.createdAt)}>
                {formatRelativeTime(credential.createdAt)}
              </span>

              <span className="text-muted-foreground">Updated</span>
              <span title={formatAbsoluteTime(credential.updatedAt)}>
                {formatRelativeTime(credential.updatedAt)}
              </span>

              {credential.url && (
                <>
                  <span className="text-muted-foreground">URL</span>
                  <span className="truncate">{credential.url}</span>
                </>
              )}

              {credential.email && (
                <>
                  <span className="text-muted-foreground">Email</span>
                  <span className="truncate">{credential.email}</span>
                </>
              )}

              {credential.description && (
                <>
                  <span className="text-muted-foreground col-span-2">Description</span>
                  <p className="col-span-2 text-sm">{credential.description}</p>
                </>
              )}
            </div>
          </div>

          {/* Bindings summary */}
          <div className="space-y-2">
            <h3 className="text-sm font-medium text-muted-foreground">Bindings</h3>
            <p className="text-sm">
              {bindingCount === 0
                ? 'Not bound to any projects or agents.'
                : `Bound to ${bindingCount} ${bindingCount === 1 ? 'target' : 'targets'}. Use the Access Matrix tab to manage bindings.`}
            </p>
          </div>

          <Separator />

          {/* Rotate Token */}
          <div className="space-y-3">
            <h3 className="text-sm font-medium">Rotate Token</h3>
            <p className="text-xs text-muted-foreground">
              Replace the existing secret with a new value. This takes effect immediately.
            </p>
            <div className="flex items-center gap-2">
              <Input
                type="password"
                placeholder="Enter new token"
                value={newToken}
                onChange={(e) => setNewToken(e.target.value)}
                autoComplete="off"
                className="flex-1"
              />
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!newToken || updateCredential.isPending}
                  >
                    {updateCredential.isPending ? 'Rotating...' : 'Rotate'}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Rotate token?</AlertDialogTitle>
                    <AlertDialogDescription>
                      This will replace the existing token for &quot;{credential.name}&quot;.
                      Any agents using this credential will immediately use the new token.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction onClick={handleRotateToken}>
                      Rotate Token
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
            {rotateError && (
              <p className="text-sm text-destructive">{rotateError}</p>
            )}
          </div>

          <Separator />

          {/* Delete */}
          <div className="space-y-3">
            <h3 className="text-sm font-medium text-destructive">Danger Zone</h3>
            <p className="text-xs text-muted-foreground">
              Permanently delete this credential. This cannot be undone.
            </p>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={deleteCredential.isPending}
                >
                  {deleteCredential.isPending ? 'Deleting...' : 'Delete Credential'}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete credential?</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will permanently delete &quot;{credential.name}&quot; and remove all
                    associated bindings. This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    onClick={handleDelete}
                    className="bg-destructive text-white hover:bg-destructive/90"
                  >
                    Delete
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
