'use client'

import { useState } from 'react'
import { KeyRound, AlertTriangle } from 'lucide-react'
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
import { toast } from 'sonner'
import { useCredential, useUpdateCredential, useDeleteCredential } from '@/queries/use-credentials'
import { useRoleBindings } from '@/queries/use-role-bindings'

export function CredentialManageSheet({
  credential,
  open,
  onOpenChange,
  onNavigateToMatrix,
}: {
  credential: DomainCredential
  open: boolean
  onOpenChange: (open: boolean) => void
  onNavigateToMatrix?: (credentialName: string) => void
}) {
  const [newToken, setNewToken] = useState('')
  const [rotateError, setRotateError] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const updateCredential = useUpdateCredential()
  const deleteCredential = useDeleteCredential()

  const { data: liveCredential } = useCredential(credential.id)
  const resolved = liveCredential ?? credential

  const safeId = credential.id.replace(/[^a-zA-Z0-9_-]/g, '')
  const { data: bindingsData } = useRoleBindings(
    { search: `credential_id = '${safeId}'` },
  )

  const providerMeta = getProviderMeta(resolved.provider)
  const category = getCategoryForProvider(resolved.provider)
  const bindingCount = bindingsData?.items.length ?? 0

  async function handleRotateToken() {
    if (!credential || !newToken) return
    setRotateError(null)

    try {
      await updateCredential.mutateAsync({
        id: credential.id,
        request: { token: newToken },
      })
      toast.success(`Token rotated for "${credential.name}"`)
      setNewToken('')
    } catch (err) {
      console.error('rotate token failed', err)
      setRotateError('Failed to rotate token. Please try again.')
    }
  }

  async function handleDelete() {
    if (!credential) return
    setDeleteError(null)
    try {
      await deleteCredential.mutateAsync(credential.id)
      toast.success(`Credential "${credential.name}" deleted`)
      onOpenChange(false)
    } catch (err) {
      console.error('delete credential failed', err)
      setDeleteError('Failed to delete credential. It may have active bindings.')
    }
  }

  function handleClose(v: boolean) {
    if (!v) {
      setNewToken('')
      setRotateError(null)
      setDeleteError(null)
    }
    onOpenChange(v)
  }

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent side="right" className="sm:max-w-lg overflow-y-auto">
        <SheetHeader className="border-l-4 border-primary pl-3">
          <SheetTitle className="flex items-center gap-2">
            <div className="size-8 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
              <KeyRound className="h-4 w-4 text-primary" />
            </div>
            <div>
              <span>{resolved.name}</span>
              <div className="flex items-center gap-2 mt-0.5">
                <Badge variant="outline" className="text-xs font-normal">
                  {providerMeta?.label ?? resolved.provider}
                </Badge>
                {category && (
                  <span className="text-xs text-muted-foreground">{category}</span>
                )}
              </div>
            </div>
          </SheetTitle>
          <SheetDescription>
            Manage credential settings and access.
          </SheetDescription>
        </SheetHeader>

        <div className="flex flex-col gap-5 px-4 pb-4">
          {/* Details */}
          <div className="rounded-lg bg-muted/40 p-4 space-y-3">
            <h3 className="text-base font-semibold tracking-tight">Details</h3>
            <div className="grid grid-cols-2 gap-x-6 gap-y-4">
              <div>
                <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Created</span>
                <p className="text-sm mt-0.5" title={formatAbsoluteTime(resolved.createdAt)}>
                  {formatRelativeTime(resolved.createdAt)}
                </p>
              </div>
              <div>
                <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Updated</span>
                <p className="text-sm mt-0.5" title={formatAbsoluteTime(resolved.updatedAt)}>
                  {formatRelativeTime(resolved.updatedAt)}
                </p>
              </div>
              {resolved.url && (
                <div className="col-span-2">
                  <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">URL</span>
                  <p className="text-sm mt-0.5 truncate">
                    {/^https?:\/\//i.test(resolved.url) ? (
                      <a href={resolved.url} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                        {resolved.url}
                      </a>
                    ) : (
                      <span className="truncate text-muted-foreground">{resolved.url}</span>
                    )}
                  </p>
                </div>
              )}
              {resolved.email && (
                <div className="col-span-2">
                  <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Email</span>
                  <p className="text-sm mt-0.5">
                    <a href={`mailto:${resolved.email}`} className="text-primary hover:underline">
                      {resolved.email}
                    </a>
                  </p>
                </div>
              )}
              {resolved.description && (
                <div className="col-span-2">
                  <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Description</span>
                  <p className="text-sm mt-0.5 text-muted-foreground">{resolved.description}</p>
                </div>
              )}
            </div>
          </div>

          {/* Bindings summary */}
          <div className="rounded-lg border p-4 space-y-2">
            <h3 className="text-base font-semibold tracking-tight">Bindings</h3>
            <p className="text-sm text-muted-foreground">
              {bindingCount === 0 ? (
                <>
                  Not bound to any projects or agents.{' '}
                  {onNavigateToMatrix && (
                    <button
                      type="button"
                      className="text-primary underline underline-offset-2 hover:text-primary/80"
                      onClick={() => {
                        onOpenChange(false)
                        onNavigateToMatrix(resolved.name)
                      }}
                    >
                      Set up access
                    </button>
                  )}
                </>
              ) : (
                <>
                  Bound to <span className="font-medium text-foreground">{bindingCount}</span> {bindingCount === 1 ? 'target' : 'targets'}.{' '}
                  {onNavigateToMatrix ? (
                    <button
                      type="button"
                      className="text-primary underline underline-offset-2 hover:text-primary/80"
                      onClick={() => {
                        onOpenChange(false)
                        onNavigateToMatrix(resolved.name)
                      }}
                    >
                      View bindings
                    </button>
                  ) : (
                    'Use the Bindings tab to manage access.'
                  )}
                </>
              )}
            </p>
          </div>

          {/* Rotate Token */}
          <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 p-4 space-y-3">
            <h3 className="text-base font-semibold tracking-tight flex items-center gap-1.5">
              <AlertTriangle className="h-4 w-4 text-amber-500" />
              Rotate Token
            </h3>
            <p className="text-xs text-muted-foreground">
              Replace the existing secret with a new value. Running sessions automatically pick up the new token within 5 minutes.
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
                      This will replace the existing token for &quot;{resolved.name}&quot;.
                      New sessions use the new token immediately. Running sessions pick it up within 5 minutes.
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

          {/* Danger Zone */}
          <div className="rounded-lg border-2 border-destructive/30 bg-destructive/5 p-4 space-y-3 mt-2">
            <h3 className="text-base font-semibold text-destructive">Danger Zone</h3>
            <p className="text-xs text-muted-foreground">
              Permanently delete this credential and revoke all bindings. Running sessions lose access within 5 minutes as sidecars refresh. This cannot be undone.
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
                    This will permanently delete &quot;{resolved.name}&quot; and remove all
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
            {deleteError && (
              <p className="text-sm text-destructive">{deleteError}</p>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
