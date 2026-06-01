'use client'

import { useCallback, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, Eye, EyeOff, Info, Plug } from 'lucide-react'
import { useConnectionStatus } from '@/hooks/use-connection-status'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

type ConfigResponse = {
  apiServerUrl: string
  customToken: boolean
  isCustomContext: boolean
  defaultApiServerUrl: string
}

function useApiServerConfig() {
  const queryClient = useQueryClient()

  const { data } = useQuery<ConfigResponse>({
    queryKey: ['config'],
    queryFn: async () => {
      const response = await fetch('/api/config')
      if (!response.ok) {
        throw new Error('Failed to fetch config')
      }
      return response.json() as Promise<ConfigResponse>
    },
    staleTime: 60_000,
  })

  const updateContext = useCallback(
    async (newUrl?: string, token?: string) => {
      const body: Record<string, string> = {}
      if (newUrl) body.apiServerUrl = newUrl
      if (token) body.customToken = token

      const response = await fetch('/api/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!response.ok) {
        throw new Error('Failed to update config')
      }
      queryClient.removeQueries({ queryKey: ['sessions'] })
      queryClient.removeQueries({ queryKey: ['projects'] })
      await queryClient.invalidateQueries({ queryKey: ['config'] })
      await queryClient.invalidateQueries({ queryKey: ['connection-status'] })
      await queryClient.refetchQueries({ queryKey: ['sessions'] })
      await queryClient.refetchQueries({ queryKey: ['projects'] })
    },
    [queryClient],
  )

  const resetContext = useCallback(async () => {
    const response = await fetch('/api/config', { method: 'DELETE' })
    if (!response.ok) {
      throw new Error('Failed to reset config')
    }
    queryClient.removeQueries({ queryKey: ['sessions'] })
    queryClient.removeQueries({ queryKey: ['projects'] })
    await queryClient.invalidateQueries({ queryKey: ['config'] })
    await queryClient.invalidateQueries({ queryKey: ['connection-status'] })
    await queryClient.refetchQueries({ queryKey: ['sessions'] })
    await queryClient.refetchQueries({ queryKey: ['projects'] })
  }, [queryClient])

  return {
    apiServerUrl: data?.apiServerUrl ?? '',
    isCustomContext: data?.isCustomContext ?? false,
    hasCustomToken: data?.customToken ?? false,
    updateContext,
    resetContext,
  }
}

function StatusDot({ status }: { status: 'connected' | 'disconnected' | 'checking' }) {
  if (status === 'connected') {
    return (
      <span
        className="inline-block size-2 rounded-full bg-status-success-foreground"
        aria-hidden="true"
      />
    )
  }

  if (status === 'disconnected') {
    return (
      <span
        className="inline-block size-2 animate-pulse rounded-full bg-status-error-foreground"
        aria-hidden="true"
      />
    )
  }

  return (
    <span
      className="inline-block size-2 rounded-full bg-muted-foreground"
      aria-hidden="true"
    />
  )
}

function statusLabel(status: 'connected' | 'disconnected' | 'checking'): string {
  switch (status) {
    case 'connected':
      return 'Connected'
    case 'disconnected':
      return 'Disconnected'
    case 'checking':
      return 'Checking...'
  }
}

export function StatusBar() {
  const router = useRouter()
  const { status } = useConnectionStatus()
  const { apiServerUrl, isCustomContext, updateContext, resetContext } =
    useApiServerConfig()

  const [popoverOpen, setPopoverOpen] = useState(false)
  const [editUrl, setEditUrl] = useState('')
  const [editToken, setEditToken] = useState('')
  const [showToken, setShowToken] = useState(false)
  const [authExpanded, setAuthExpanded] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const [announcement, setAnnouncement] = useState('')

  const announce = useCallback((message: string) => {
    setAnnouncement('')
    requestAnimationFrame(() => setAnnouncement(message))
  }, [])

  const handleOpenChange = useCallback(
    (open: boolean) => {
      if (open) {
        setEditUrl(apiServerUrl)
        setEditToken('')
        setShowToken(false)
        setAuthExpanded(false)
      }
      setPopoverOpen(open)
      if (!open) {
        requestAnimationFrame(() => triggerRef.current?.focus())
      }
    },
    [apiServerUrl],
  )

  const handleConnect = useCallback(async () => {
    const trimmedUrl = editUrl.trim()
    const trimmedToken = editToken.trim()

    const urlChanged = trimmedUrl && trimmedUrl !== apiServerUrl
    const tokenProvided = trimmedToken.length > 0

    if (urlChanged || tokenProvided) {
      await updateContext(
        urlChanged ? trimmedUrl : undefined,
        tokenProvided ? trimmedToken : undefined,
      )
      announce('Connection updated')
      router.push('/')
    }
    setPopoverOpen(false)
    setEditToken('')
    setShowToken(false)
    requestAnimationFrame(() => triggerRef.current?.focus())
  }, [editUrl, editToken, apiServerUrl, updateContext, announce])

  const handleCancel = useCallback(() => {
    setPopoverOpen(false)
    setEditUrl('')
    setEditToken('')
    setShowToken(false)
    requestAnimationFrame(() => triggerRef.current?.focus())
  }, [])

  const handleRestore = useCallback(async () => {
    await resetContext()
    announce('Connection restored to default')
    router.push('/')
    setPopoverOpen(false)
    requestAnimationFrame(() => triggerRef.current?.focus())
  }, [resetContext, announce])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        void handleConnect()
      } else if (e.key === 'Escape') {
        e.preventDefault()
        handleCancel()
      }
    },
    [handleConnect, handleCancel],
  )

  const displayLabel = isCustomContext ? 'Custom Server' : apiServerUrl

  return (
    <div
      className="sticky bottom-0 z-30 flex h-7 items-center justify-between border-t bg-background px-3"
      aria-label="Connection status"
    >
      <span className="sr-only" aria-live="polite">
        {announcement}
      </span>

      <div className="flex items-center gap-1.5">
        <StatusDot status={status} />
        <span
          className={cn(
            'text-xs',
            status === 'connected' && 'text-status-success-foreground',
            status === 'disconnected' && 'text-status-error-foreground',
            status === 'checking' && 'text-muted-foreground',
          )}
        >
          {statusLabel(status)}
        </span>
      </div>

      <Popover open={popoverOpen} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          <button
            ref={triggerRef}
            type="button"
            className={cn(
              'flex h-6 cursor-pointer items-center gap-1.5 rounded px-1.5 text-xs hover:bg-accent',
              isCustomContext
                ? 'text-blue-500'
                : 'font-mono text-muted-foreground',
            )}
            aria-label={
              isCustomContext
                ? 'Custom server connection. Click to edit.'
                : `API server: ${apiServerUrl}. Click to edit.`
            }
          >
            {isCustomContext && <Plug className="size-3.5" />}
            <span>{displayLabel}</span>
            <ChevronDown className="size-3 opacity-60" />
          </button>
        </PopoverTrigger>

        <PopoverContent
          side="top"
          align="end"
          className="w-80"
          onOpenAutoFocus={(e) => {
            e.preventDefault()
          }}
        >
          <div className="space-y-4">
            <h3 className="text-sm font-medium">Connection</h3>

            <div className="space-y-2">
              <label
                htmlFor="status-bar-url"
                className="text-xs font-medium text-muted-foreground"
              >
                API Server URL
              </label>
              <Input
                id="status-bar-url"
                type="text"
                value={editUrl}
                onChange={(e) => setEditUrl(e.target.value)}
                onKeyDown={handleKeyDown}
                className="h-8 font-mono text-xs"
                placeholder="https://api-server:8000"
              />
            </div>

            <div className="space-y-2">
              <button
                type="button"
                className="flex cursor-pointer items-center gap-1 text-xs font-medium text-muted-foreground hover:text-foreground"
                onClick={() => setAuthExpanded((prev) => !prev)}
                aria-expanded={authExpanded}
                aria-controls="status-bar-auth-section"
              >
                <ChevronDown
                  className={cn(
                    'size-3 transition-transform',
                    !authExpanded && '-rotate-90',
                  )}
                />
                Authentication
              </button>

              {authExpanded && (
                <div id="status-bar-auth-section" className="space-y-2">
                  <label
                    htmlFor="status-bar-token"
                    className="text-xs font-medium text-muted-foreground"
                  >
                    Bearer Token
                  </label>
                  <div className="relative">
                    <Input
                      id="status-bar-token"
                      type={showToken ? 'text' : 'password'}
                      value={editToken}
                      onChange={(e) => setEditToken(e.target.value)}
                      onKeyDown={handleKeyDown}
                      className="h-8 pr-9 font-mono text-xs"
                      placeholder="Token (optional)"
                      autoComplete="off"
                      spellCheck={false}
                    />
                    <button
                      type="button"
                      className="absolute right-0 top-0 flex h-8 w-8 cursor-pointer items-center justify-center text-muted-foreground hover:text-foreground"
                      onClick={() => setShowToken((prev) => !prev)}
                      aria-label={showToken ? 'Hide token' : 'Show token'}
                      aria-pressed={showToken}
                    >
                      {showToken ? (
                        <EyeOff className="size-3.5" />
                      ) : (
                        <Eye className="size-3.5" />
                      )}
                    </button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Leave blank to use SSO credentials.
                  </p>
                </div>
              )}
            </div>

            <div className="flex justify-end gap-2">
              <Button
                variant="ghost"
                size="sm"
                className="cursor-pointer"
                onClick={handleCancel}
              >
                Cancel
              </Button>
              <Button
                size="sm"
                className="cursor-pointer"
                onClick={() => void handleConnect()}
              >
                Connect
              </Button>
            </div>

            {isCustomContext && (
              <div className="rounded-md bg-blue-50 px-3 py-2 dark:bg-blue-950/30">
                <div className="flex items-start gap-2">
                  <Info className="mt-0.5 size-3.5 shrink-0 text-blue-500" />
                  <div className="space-y-1.5">
                    <p className="text-xs text-blue-500">
                      Using a custom server.
                    </p>
                    <Button
                      variant="outline"
                      size="xs"
                      className="cursor-pointer"
                      onClick={() => void handleRestore()}
                    >
                      Restore Default
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  )
}
