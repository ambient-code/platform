'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Loader2, Eye, EyeOff } from 'lucide-react'
import { successToast, errorToast } from '@/hooks/use-toast'
import { useConnectJira, useDisconnectJira } from '@/services/queries/use-jira'

type Props = {
  status?: {
    connected: boolean
    url?: string
    email?: string
    updatedAt?: string
    valid?: boolean
  }
  onRefresh?: () => void
}

export function JiraConnectionCard({ status, onRefresh }: Props) {
  const connectMutation = useConnectJira()
  const disconnectMutation = useDisconnectJira()
  const isLoading = !status
  
  const [showForm, setShowForm] = useState(false)
  const [url, setUrl] = useState('')
  const [email, setEmail] = useState('')
  const [apiToken, setApiToken] = useState('')
  const [showToken, setShowToken] = useState(false)

  const handleConnect = async () => {
    if (!url || !email || !apiToken) {
      errorToast('Please fill in all fields')
      return
    }

    connectMutation.mutate(
      { url, email, apiToken },
      {
        onSuccess: () => {
          successToast('Jira connected successfully')
          setShowForm(false)
          setUrl('')
          setEmail('')
          setApiToken('')
          onRefresh?.()
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to connect Jira')
        },
      }
    )
  }

  const handleDisconnect = async () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        successToast('Jira disconnected successfully')
        onRefresh?.()
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to disconnect Jira')
      },
    })
  }

  const handleEdit = () => {
    // Pre-fill form with existing values for editing
    if (status?.connected) {
      setUrl(status.url || '')
      setEmail(status.email || '')
      setShowForm(true)
    }
  }

  return (
    <Card className="bg-card border border-gray-200 shadow-sm flex flex-col h-full">
      <div className="p-6 flex flex-col flex-1">
        {/* Header section with icon and title */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-blue-600 rounded-lg flex items-center justify-center">
            <svg className="w-10 h-10 text-white" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M11.571 11.513H0a5.218 5.218 0 0 0 5.232 5.215h2.13v2.057A5.215 5.215 0 0 0 12.575 24V12.518a1.005 1.005 0 0 0-1.005-1.005zm5.723-5.756H5.736a5.215 5.215 0 0 0 5.215 5.214h2.129v2.058a5.218 5.218 0 0 0 5.215 5.214V6.758a1.001 1.001 0 0 0-1.001-1.001zM23.013 0H11.455a5.215 5.215 0 0 0 5.215 5.215h2.129v2.057A5.215 5.215 0 0 0 24 12.483V1.005A1.001 1.001 0 0 0 23.013 0z" />
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">Jira</h3>
            <p className="text-muted-foreground">Connect to Jira for issue tracking</p>
          </div>
        </div>

        {/* Status section */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span className={`w-2 h-2 rounded-full ${status?.connected && status.valid !== false ? 'bg-green-500' : status?.connected ? 'bg-yellow-500' : 'bg-gray-400'}`}></span>
            <span className="text-sm font-medium text-foreground/80">
              {status?.connected ? (
                <>Connected{status.email ? ` as ${status.email}` : ''}</>
              ) : (
                'Not Connected'
              )}
            </span>
          </div>
          {status?.connected && status.valid === false && (
            <p className="text-xs text-yellow-600 dark:text-yellow-400 mb-2">
              ⚠️ Token appears invalid - click Edit to update
            </p>
          )}
          {status?.connected && status.url && (
            <p className="text-sm text-muted-foreground mb-2">
              Instance: {status.url}
            </p>
          )}
          <p className="text-muted-foreground">
            Connect to Jira to search issues, create tickets, and track work across all sessions
          </p>
        </div>

        {/* Connection form */}
        {showForm && (
          <div className="mb-4 space-y-3">
            <div>
              <Label htmlFor="jira-url" className="text-sm">Jira URL</Label>
              <Input
                id="jira-url"
                type="url"
                placeholder="https://your-company.atlassian.net"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                disabled={connectMutation.isPending}
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="jira-email" className="text-sm">Email</Label>
              <Input
                id="jira-email"
                type="email"
                placeholder="your-email@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={connectMutation.isPending}
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="jira-token" className="text-sm">API Token</Label>
              <div className="flex gap-2 mt-1">
                <Input
                  id="jira-token"
                  type={showToken ? 'text' : 'password'}
                  placeholder="Your Jira API token"
                  value={apiToken}
                  onChange={(e) => setApiToken(e.target.value)}
                  disabled={connectMutation.isPending}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowToken(!showToken)}
                  disabled={connectMutation.isPending}
                >
                  {showToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                Create an API token at{' '}
                <a
                  href={url ? `${url}/secure/ViewProfile.jspa?selectedTab=com.atlassian.pats.pats-plugin:jira-user-personal-access-tokens` : 'https://id.atlassian.com/manage-profile/security/api-tokens'}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline"
                >
                  Jira Settings
                </a>
              </p>
            </div>
            <div className="flex gap-2 pt-2">
              <Button
                onClick={handleConnect}
                disabled={connectMutation.isPending || !url || !email || !apiToken}
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
              className="bg-blue-600 hover:bg-blue-700 text-white"
            >
              Connect Jira
            </Button>
          ) : null}
        </div>
      </div>
    </Card>
  )
}
