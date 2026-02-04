'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Loader2, Eye, EyeOff } from 'lucide-react'
import { successToast, errorToast } from '@/hooks/use-toast'
import { useConnectGitLab, useDisconnectGitLab } from '@/services/queries/use-gitlab'

type Props = {
  status?: {
    connected: boolean
    instanceUrl?: string
    updatedAt?: string
    valid?: boolean
  }
  onRefresh?: () => void
}

export function GitLabConnectionCard({ status, onRefresh }: Props) {
  const connectMutation = useConnectGitLab()
  const disconnectMutation = useDisconnectGitLab()
  const isLoading = !status
  
  const [showForm, setShowForm] = useState(false)
  const [instanceUrl, setInstanceUrl] = useState('https://gitlab.com')
  const [token, setToken] = useState('')
  const [showToken, setShowToken] = useState(false)

  const handleConnect = async () => {
    if (!token) {
      errorToast('Please enter your GitLab Personal Access Token')
      return
    }

    connectMutation.mutate(
      { personalAccessToken: token, instanceUrl },
      {
        onSuccess: () => {
          successToast('GitLab connected successfully')
          setShowForm(false)
          setToken('')
          onRefresh?.()
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to connect GitLab')
        },
      }
    )
  }

  const handleDisconnect = async () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        successToast('GitLab disconnected successfully')
        onRefresh?.()
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to disconnect GitLab')
      },
    })
  }

  const handleEdit = () => {
    // Pre-fill form with existing values for editing
    if (status?.connected) {
      setInstanceUrl(status.instanceUrl || 'https://gitlab.com')
      setShowForm(true)
    }
  }

  return (
    <Card className="bg-card border border-gray-200 shadow-sm flex flex-col h-full">
      <div className="p-6 flex flex-col flex-1">
        {/* Header section with icon and title */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-orange-600 rounded-lg flex items-center justify-center">
            <svg className="w-10 h-10 text-white" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M23.6004 9.5927l-.0337-.0862L20.3.9814a.851.851 0 00-.3362-.405.8748.8748 0 00-.9997.0539.8748.8748 0 00-.29.4399l-2.2055 6.748H7.5375l-2.2057-6.748a.8573.8573 0 00-.29-.4412.8748.8748 0 00-.9997-.0537.8585.8585 0 00-.3362.4049L.4332 9.5015l-.0325.0862a6.0657 6.0657 0 002.0119 7.0105l.0113.0087.03.0213 4.976 3.7264 2.462 1.8633 1.4995 1.1321a1.0085 1.0085 0 001.2197 0l1.4995-1.1321 2.4619-1.8633 5.006-3.7489.0125-.01a6.0682 6.0682 0 002.0094-7.003z"/>
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">GitLab</h3>
            <p className="text-muted-foreground">Connect to GitLab repositories</p>
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
              ⚠️ Token appears invalid - click Edit to update
            </p>
          )}
          {status?.connected && status.instanceUrl && (
            <p className="text-sm text-muted-foreground mb-2">
              Instance: {status.instanceUrl}
            </p>
          )}
          <p className="text-muted-foreground">
            Connect to GitLab to clone, commit, and push to repositories across all sessions
          </p>
        </div>

        {/* Connection form */}
        {showForm && (
          <div className="mb-4 space-y-3">
            <div>
              <Label htmlFor="gitlab-url" className="text-sm">GitLab Instance URL</Label>
              <Input
                id="gitlab-url"
                type="url"
                placeholder="https://gitlab.com"
                value={instanceUrl}
                onChange={(e) => setInstanceUrl(e.target.value)}
                disabled={connectMutation.isPending}
                className="mt-1"
              />
              <p className="text-xs text-muted-foreground mt-1">
                Use https://gitlab.com for GitLab.com or your self-hosted instance URL
              </p>
            </div>
            <div>
              <Label htmlFor="gitlab-token" className="text-sm">Personal Access Token</Label>
              <div className="flex gap-2 mt-1">
                <Input
                  id="gitlab-token"
                  type={showToken ? 'text' : 'password'}
                  placeholder="glpat-xxxxxxxxxxxxxxxxxxxx"
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
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
                Create a token with <code>api</code>, <code>read_api</code>, <code>read_user</code>, and{' '}
                <code>write_repository</code> scopes at{' '}
                <a
                  href={`${instanceUrl}/-/user_settings/personal_access_tokens`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline"
                >
                  GitLab Settings
                </a>
              </p>
            </div>
            <div className="flex gap-2 pt-2">
              <Button
                onClick={handleConnect}
                disabled={connectMutation.isPending || !token}
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
              Connect GitLab
            </Button>
          ) : null}
        </div>
      </div>
    </Card>
  )
}
