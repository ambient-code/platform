'use client'

import { useState, useCallback, useRef } from 'react'
import { KeyRound, AlertTriangle, Eye, EyeOff, Upload, X } from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
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
  const [showRotateSecret, setShowRotateSecret] = useState(false)
  const [rotateFileName, setRotateFileName] = useState<string | null>(null)
  const rotateFileRef = useRef<HTMLInputElement>(null)
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
      setRotateFileName(null)
      setShowRotateSecret(false)
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

  const MAX_UPLOAD_BYTES = 1_048_576

  const handleRotateFileUpload = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    if (file.size > MAX_UPLOAD_BYTES) {
      setRotateError('File exceeds 1 MB limit.')
      return
    }
    const reader = new FileReader()
    reader.onload = () => {
      const text = reader.result
      if (typeof text === 'string') {
        setNewToken(text)
        setRotateFileName(file.name)
        setRotateError(null)
      }
    }
    reader.onerror = () => {
      setRotateError('Failed to read file.')
      setNewToken('')
      setRotateFileName(null)
    }
    reader.readAsText(file)
  }, [])

  function handleClose(v: boolean) {
    if (!v) {
      setNewToken('')
      setRotateError(null)
      setDeleteError(null)
      setShowRotateSecret(false)
      setRotateFileName(null)
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
              Rotate {providerMeta?.tokenField?.label ?? 'Token'}
            </h3>
            <p className="text-xs text-muted-foreground">
              Replace the existing secret with a new value. New sessions use it immediately. Restart running sessions to pick up the change.
            </p>
            {providerMeta?.tokenField?.multiline ? (
              <div className="space-y-2">
                {rotateFileName ? (
                  <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-2 text-sm">
                    <Upload className="size-4 text-muted-foreground shrink-0" />
                    <span className="truncate flex-1 font-mono text-xs">{rotateFileName}</span>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="size-6 shrink-0"
                      onClick={() => { setNewToken(''); setRotateFileName(null) }}
                      aria-label="Remove uploaded file"
                    >
                      <X className="size-3.5" />
                    </Button>
                  </div>
                ) : (
                  <div className="relative">
                    <Textarea
                      placeholder={providerMeta.tokenField.placeholder}
                      value={newToken}
                      onChange={(e) => setNewToken(e.target.value)}
                      autoComplete="off"
                      className={`min-h-24 font-mono text-xs pr-10`}
                      style={!showRotateSecret ? { WebkitTextSecurity: 'disc' } as React.CSSProperties : undefined}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-1.5 top-1.5 size-7"
                      onClick={() => setShowRotateSecret((v) => !v)}
                      aria-label={showRotateSecret ? 'Hide secret' : 'Show secret'}
                    >
                      {showRotateSecret ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                    </Button>
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-7 text-xs"
                    onClick={() => rotateFileRef.current?.click()}
                  >
                    <Upload className="size-3 mr-1" />
                    Upload file
                  </Button>
                  <input
                    ref={rotateFileRef}
                    type="file"
                    className="hidden"
                    accept=".yaml,.yml,.json,.txt,.pem,.key,*"
                    onChange={handleRotateFileUpload}
                  />
                  <div className="flex-1" />
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7"
                        disabled={!newToken || updateCredential.isPending}
                      >
                        {updateCredential.isPending ? 'Rotating...' : 'Rotate'}
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Rotate {providerMeta.tokenField.label.toLowerCase()}?</AlertDialogTitle>
                        <AlertDialogDescription>
                          This will replace the existing {providerMeta.tokenField.label.toLowerCase()} for &quot;{resolved.name}&quot;.
                          New sessions use the new value immediately. Restart running sessions to pick up the change.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={handleRotateToken}>
                          Rotate
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <div className="relative flex-1">
                  <Input
                    type={showRotateSecret ? 'text' : 'password'}
                    placeholder="Enter new token"
                    value={newToken}
                    onChange={(e) => setNewToken(e.target.value)}
                    autoComplete="off"
                    className="pr-10"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-1 top-1/2 -translate-y-1/2 size-7"
                    onClick={() => setShowRotateSecret((v) => !v)}
                    aria-label={showRotateSecret ? 'Hide token' : 'Show token'}
                  >
                    {showRotateSecret ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                  </Button>
                </div>
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
                        New sessions use the new token immediately. Restart running sessions to pick up the change.
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
            )}
            {rotateError && (
              <p className="text-sm text-destructive">{rotateError}</p>
            )}
          </div>

          {/* Danger Zone */}
          <div className="rounded-lg border-2 border-destructive/30 bg-destructive/5 p-4 space-y-3 mt-2">
            <h3 className="text-base font-semibold text-destructive">Danger Zone</h3>
            <p className="text-xs text-muted-foreground">
              Permanently delete this credential and revoke all bindings. Running sessions keep access until restarted. This cannot be undone.
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
