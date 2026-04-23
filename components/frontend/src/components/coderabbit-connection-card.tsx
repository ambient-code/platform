'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Loader2, Eye, EyeOff, ChevronDown, ChevronRight, TriangleAlert } from 'lucide-react'
import { toast } from 'sonner'
import { useConnectCodeRabbit, useDisconnectCodeRabbit } from '@/services/queries/use-coderabbit'

type Props = {
  status?: {
    connected: boolean
    updatedAt?: string
    valid?: boolean
  }
  onRefresh?: () => void
}

export function CodeRabbitConnectionCard({ status, onRefresh }: Props) {
  const connectMutation = useConnectCodeRabbit()
  const disconnectMutation = useDisconnectCodeRabbit()
  const isLoading = !status

  const [showAdvanced, setShowAdvanced] = useState(status?.connected ?? false)
  const [showForm, setShowForm] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [showKey, setShowKey] = useState(false)

  const handleConnect = async () => {
    if (!apiKey) {
      toast.error('Please enter an API key')
      return
    }

    connectMutation.mutate(
      { apiKey },
      {
        onSuccess: () => {
          toast.success('API key saved for private repository access')
          setShowForm(false)
          setApiKey('')
          onRefresh?.()
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to save API key')
        },
      }
    )
  }

  const handleDisconnect = async () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        toast.success('API key removed')
        onRefresh?.()
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Failed to remove API key')
      },
    })
  }

  return (
    <Card className="bg-card border border-border/60 shadow-sm shadow-black/[0.03] dark:shadow-black/[0.15] flex flex-col h-full">
      <div className="p-6 flex flex-col flex-1">
        {/* Header */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-primary rounded-lg flex items-center justify-center">
            <svg className="w-10 h-10 text-white" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 2L2 7v10c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V7l-10-5zm0 18c-3.87-.96-7-5.15-7-9V8.17l7-3.5 7 3.5V11c0 3.85-3.13 8.04-7 9z"/>
              <path d="M10 14l-2-2 1.41-1.41L10 11.17l3.59-3.58L15 9l-5 5z"/>
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">CodeRabbit</h3>
            <p className="text-muted-foreground">AI-powered code review</p>
          </div>
        </div>

        {/* Default status — public repos are free */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span className="w-2 h-2 rounded-full bg-green-500" />
            <span className="text-sm font-medium text-foreground/80">
              Active for public repositories
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            Code review is free for public repositories via the{' '}
            <a
              href="https://github.com/apps/coderabbitai"
              target="_blank"
              rel="noopener noreferrer"
              className="underline"
            >
              CodeRabbit GitHub App
            </a>
            . No configuration needed.
          </p>
        </div>

        {/* Private repo access — collapsed by default */}
        <div className="mt-auto">
          <Button
            type="button"
            variant="ghost"
            className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground h-auto p-0"
            onClick={() => setShowAdvanced(!showAdvanced)}
          >
            {showAdvanced ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            Private repository access
            {status?.connected && (
              <span className="ml-1.5 w-2 h-2 rounded-full bg-green-500 inline-block" />
            )}
          </Button>

          {showAdvanced && (
            <div className="mt-3 pt-3 border-t border-border/40 space-y-3">
              {/* Billing warning */}
              <div className="flex gap-2 p-2.5 rounded-md bg-amber-500/10 border border-amber-500/20">
                <TriangleAlert className="h-4 w-4 text-amber-500 flex-shrink-0 mt-0.5" />
                <p className="text-xs text-amber-700 dark:text-amber-400">
                  Only needed for private repositories. Using an API key on public repos will incur
                  charges for reviews that are otherwise free.
                </p>
              </div>

              {status?.connected ? (
                <>
                  <p className="text-sm text-muted-foreground">
                    API key configured for private repository reviews.
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setShowForm(true)}
                      disabled={isLoading || disconnectMutation.isPending}
                    >
                      Update Key
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={handleDisconnect}
                      disabled={isLoading || disconnectMutation.isPending}
                    >
                      {disconnectMutation.isPending ? (
                        <>
                          <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />
                          Removing...
                        </>
                      ) : (
                        'Remove Key'
                      )}
                    </Button>
                  </div>
                </>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Add an API key to enable CLI reviews for private repositories in sessions.
                </p>
              )}

              {/* API key form */}
              {(showForm || (!status?.connected && showAdvanced)) && (
                <div className="space-y-3">
                  <div>
                    <Label htmlFor="coderabbit-key" className="text-sm">API Key</Label>
                    <div className="flex gap-2 mt-1">
                      <Input
                        id="coderabbit-key"
                        type={showKey ? 'text' : 'password'}
                        placeholder="cr-..."
                        value={apiKey}
                        onChange={(e) => setApiKey(e.target.value)}
                        disabled={connectMutation.isPending}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowKey(!showKey)}
                        disabled={connectMutation.isPending}
                      >
                        {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </Button>
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      Log in with GitHub at{' '}
                      <a
                        href="https://app.coderabbit.ai/settings/api-keys"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="underline"
                      >
                        CodeRabbit API Keys
                      </a>
                      {' '}to generate a key.
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      onClick={handleConnect}
                      disabled={connectMutation.isPending || !apiKey}
                    >
                      {connectMutation.isPending ? (
                        <>
                          <Loader2 className="h-3 w-3 mr-1.5 animate-spin" />
                          Saving...
                        </>
                      ) : (
                        'Save Key'
                      )}
                    </Button>
                    {status?.connected && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setShowForm(false)}
                        disabled={connectMutation.isPending}
                      >
                        Cancel
                      </Button>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </Card>
  )
}
