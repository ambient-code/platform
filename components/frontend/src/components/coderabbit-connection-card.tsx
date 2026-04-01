'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Loader2, Eye, EyeOff } from 'lucide-react'
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

  const [showForm, setShowForm] = useState(false)
  const [apiKey, setApiKey] = useState('')
  const [showApiKey, setShowApiKey] = useState(false)

  const handleConnect = async () => {
    if (!apiKey) {
      toast.error('Please enter an API key')
      return
    }

    connectMutation.mutate(
      { apiKey },
      {
        onSuccess: () => {
          toast.success('CodeRabbit connected successfully')
          setShowForm(false)
          setApiKey('')
          onRefresh?.()
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to connect CodeRabbit')
        },
      }
    )
  }

  const handleDisconnect = async () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        toast.success('CodeRabbit disconnected successfully')
        onRefresh?.()
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Failed to disconnect CodeRabbit')
      },
    })
  }

  const handleEdit = () => {
    setShowForm(true)
  }

  return (
    <Card className="bg-card border border-border/60 shadow-sm shadow-black/[0.03] dark:shadow-black/[0.15] flex flex-col h-full">
      <div className="p-6 flex flex-col flex-1">
        {/* Header section with icon and title */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-primary rounded-lg flex items-center justify-center">
            <svg className="w-10 h-10 text-white" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">CodeRabbit</h3>
            <p className="text-muted-foreground">Connect to CodeRabbit for AI-powered code review</p>
          </div>
        </div>

        {/* Status section */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span className={`w-2 h-2 rounded-full ${status?.connected && status.valid !== false ? 'bg-green-500' : status?.connected ? 'bg-yellow-500' : 'bg-gray-400'}`}></span>
            <span className="text-sm font-medium text-foreground/80">
              {status?.connected ? 'Connected' : 'Not Connected'}
            </span>
          </div>
          {status?.connected && status.valid === false && (
            <p className="text-xs text-yellow-600 dark:text-yellow-400 mb-2">
              ⚠️ API key appears invalid - click Edit to update
            </p>
          )}
          {status?.connected && status.updatedAt && (
            <p className="text-sm text-muted-foreground mb-2">
              Last updated: {new Date(status.updatedAt).toLocaleString()}
            </p>
          )}
          <p className="text-muted-foreground">
            Connect to CodeRabbit to enable AI-powered code review across all sessions
          </p>
        </div>

        {/* Connection form */}
        {showForm && (
          <div className="mb-4 space-y-3">
            <div>
              <Label htmlFor="coderabbit-api-key" className="text-sm">API Key</Label>
              <div className="flex gap-2 mt-1">
                <Input
                  id="coderabbit-api-key"
                  type={showApiKey ? 'text' : 'password'}
                  placeholder="Your CodeRabbit API key"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  disabled={connectMutation.isPending}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowApiKey(!showApiKey)}
                  disabled={connectMutation.isPending}
                >
                  {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                Create an API key at{' '}
                <a
                  href="https://app.coderabbit.ai/settings/api-keys"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline"
                >
                  CodeRabbit Settings
                </a>
              </p>
            </div>
            <div className="flex gap-2 pt-2">
              <Button
                onClick={handleConnect}
                disabled={connectMutation.isPending || !apiKey}
                className="flex-1"
              >
                {connectMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Connecting...
                  </>
                ) : (
                  'Save Credentials'
                )}
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowForm(false)}
                disabled={connectMutation.isPending}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        {/* Action buttons */}
        <div className="flex gap-3 mt-auto">
          {status?.connected && !showForm ? (
            <>
              <Button
                variant="outline"
                onClick={handleEdit}
                disabled={isLoading || disconnectMutation.isPending}
              >
                Edit
              </Button>
              <Button
                variant="destructive"
                onClick={handleDisconnect}
                disabled={isLoading || disconnectMutation.isPending}
              >
                {disconnectMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Disconnecting...
                  </>
                ) : (
                  'Disconnect'
                )}
              </Button>
            </>
          ) : !showForm ? (
            <Button
              onClick={() => setShowForm(true)}
              disabled={isLoading}
              className="bg-primary hover:bg-primary/90 text-primary-foreground"
            >
              Connect CodeRabbit
            </Button>
          ) : null}
        </div>
      </div>
    </Card>
  )
}
