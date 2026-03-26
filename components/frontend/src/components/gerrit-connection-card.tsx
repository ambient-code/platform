'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Loader2, Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'
import { useConnectGerrit, useDisconnectGerrit, useTestGerritConnection } from '@/services/queries/use-gerrit'
import type { GerritAuthMethod } from '@/services/api/gerrit-auth'

type GerritInstanceInfo = {
  connected: boolean
  instanceName: string
  url: string
  authMethod: 'http_basic' | 'git_cookies'
  updatedAt: string
}

type Props = {
  status?: {
    instances: GerritInstanceInfo[]
  }
  onRefresh?: () => void
}

/** Card component for managing Gerrit instance connections with multi-instance support. */
export function GerritConnectionCard({ status, onRefresh }: Props) {
  const connectMutation = useConnectGerrit()
  const disconnectMutation = useDisconnectGerrit()
  const testMutation = useTestGerritConnection()
  const isLoading = !status

  const [showForm, setShowForm] = useState(false)
  const [instanceName, setInstanceName] = useState('')
  const [url, setUrl] = useState('')
  const [authMethod, setAuthMethod] = useState<GerritAuthMethod>('http_basic')
  const [username, setUsername] = useState('')
  const [httpToken, setHttpToken] = useState('')
  const [gitcookiesContent, setGitcookiesContent] = useState('')
  const [showToken, setShowToken] = useState(false)

  const instances = status?.instances ?? []
  const hasInstances = instances.length > 0

  const resetForm = () => {
    setInstanceName('')
    setUrl('')
    setAuthMethod('http_basic')
    setUsername('')
    setHttpToken('')
    setGitcookiesContent('')
    setShowToken(false)
    setShowForm(false)
  }

  const buildRequest = () => ({
    instanceName,
    url,
    authMethod,
    ...(authMethod === 'http_basic' ? { username, httpToken } : { gitcookiesContent }),
  })

  const isFormValid = () => {
    if (!instanceName || instanceName.length < 2 || !url) return false
    if (authMethod === 'http_basic') return !!username && !!httpToken
    return !!gitcookiesContent
  }

  const handleTest = () => {
    testMutation.mutate(
      { url, authMethod, ...(authMethod === 'http_basic' ? { username, httpToken } : { gitcookiesContent }) },
      {
        onSuccess: (result) => {
          if (result.valid) {
            toast.success('Connection test successful')
          } else {
            toast.error(result.error || 'Connection test failed')
          }
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Connection test failed')
        },
      }
    )
  }

  const handleConnect = () => {
    connectMutation.mutate(buildRequest(), {
      onSuccess: () => {
        toast.success(`Gerrit instance '${instanceName}' connected successfully`)
        resetForm()
        onRefresh?.()
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Failed to connect Gerrit')
      },
    })
  }

  const handleDisconnect = (name: string) => {
    disconnectMutation.mutate(name, {
      onSuccess: () => {
        toast.success(`Gerrit instance '${name}' disconnected`)
        onRefresh?.()
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Failed to disconnect')
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
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z" />
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">Gerrit</h3>
            <p className="text-muted-foreground">Connect to Gerrit for code review</p>
          </div>
        </div>

        {/* Connected instances list */}
        {hasInstances && !showForm && (
          <div className="mb-4 space-y-2">
            {instances.map((inst) => (
              <div key={inst.instanceName} className="flex items-center justify-between p-3 rounded-md bg-muted/50 border border-border/40">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="w-2 h-2 rounded-full bg-green-500 flex-shrink-0"></span>
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">{inst.instanceName}</p>
                    <p className="text-xs text-muted-foreground truncate">{inst.url}</p>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDisconnect(inst.instanceName)}
                  disabled={disconnectMutation.isPending}
                  className="text-destructive hover:text-destructive flex-shrink-0"
                >
                  Disconnect
                </Button>
              </div>
            ))}
          </div>
        )}

        {/* Status when no instances */}
        {!hasInstances && !showForm && (
          <div className="mb-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="w-2 h-2 rounded-full bg-gray-400"></span>
              <span className="text-sm font-medium text-foreground/80">Not Connected</span>
            </div>
            <p className="text-muted-foreground">
              Connect to Gerrit to review changes, submit comments, and manage code reviews across all sessions
            </p>
          </div>
        )}

        {/* Connection form */}
        {showForm && (
          <div className="mb-4 space-y-3">
            <div>
              <Label htmlFor="gerrit-instance-name" className="text-sm">Instance Name</Label>
              <Input
                id="gerrit-instance-name"
                type="text"
                placeholder="e.g. openstack, android"
                value={instanceName}
                onChange={(e) => setInstanceName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
                disabled={connectMutation.isPending}
                className="mt-1"
                minLength={2}
                maxLength={63}
              />
              <p className="text-xs text-muted-foreground mt-1">
                A short name to identify this Gerrit instance (2-63 chars, lowercase, hyphens allowed)
              </p>
            </div>
            <div>
              <Label htmlFor="gerrit-url" className="text-sm">Gerrit URL</Label>
              <Input
                id="gerrit-url"
                type="url"
                placeholder="https://review.opendev.org"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                disabled={connectMutation.isPending}
                className="mt-1"
              />
            </div>
            <div>
              <Label className="text-sm">Authentication Method</Label>
              <div className="flex gap-4 mt-2">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="gerrit-auth-method"
                    value="http_basic"
                    checked={authMethod === 'http_basic'}
                    onChange={() => setAuthMethod('http_basic')}
                    disabled={connectMutation.isPending}
                    className="accent-primary"
                  />
                  <span className="text-sm">HTTP Credentials</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="gerrit-auth-method"
                    value="git_cookies"
                    checked={authMethod === 'git_cookies'}
                    onChange={() => setAuthMethod('git_cookies')}
                    disabled={connectMutation.isPending}
                    className="accent-primary"
                  />
                  <span className="text-sm">Gitcookies</span>
                </label>
              </div>
            </div>

            {authMethod === 'http_basic' ? (
              <>
                <div>
                  <Label htmlFor="gerrit-username" className="text-sm">Username</Label>
                  <Input
                    id="gerrit-username"
                    type="text"
                    placeholder="Your Gerrit username"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    disabled={connectMutation.isPending}
                    className="mt-1"
                  />
                </div>
                <div>
                  <Label htmlFor="gerrit-token" className="text-sm">HTTP Password</Label>
                  <div className="flex gap-2 mt-1">
                    <Input
                      id="gerrit-token"
                      type={showToken ? 'text' : 'password'}
                      placeholder="Your Gerrit HTTP password"
                      value={httpToken}
                      onChange={(e) => setHttpToken(e.target.value)}
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
                    Generate an HTTP password in Gerrit Settings &gt; HTTP Credentials
                  </p>
                </div>
              </>
            ) : (
              <div>
                <Label htmlFor="gerrit-gitcookies" className="text-sm">Gitcookies Content</Label>
                <textarea
                  id="gerrit-gitcookies"
                  className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 font-mono"
                  rows={3}
                  placeholder="Paste your .gitcookies file content here"
                  value={gitcookiesContent}
                  onChange={(e) => setGitcookiesContent(e.target.value)}
                  disabled={connectMutation.isPending}
                />
                <p className="text-xs text-muted-foreground mt-1">
                  Paste the content of your ~/.gitcookies file
                </p>
              </div>
            )}

            <div className="flex gap-2 pt-2">
              <Button
                onClick={handleConnect}
                disabled={connectMutation.isPending || !isFormValid()}
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
                onClick={handleTest}
                disabled={testMutation.isPending || !url || (authMethod === 'http_basic' ? (!username || !httpToken) : !gitcookiesContent)}
              >
                {testMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  'Test'
                )}
              </Button>
              <Button
                variant="outline"
                onClick={resetForm}
                disabled={connectMutation.isPending}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        {/* Action buttons */}
        <div className="flex gap-3 mt-auto">
          {!showForm && (
            <Button
              onClick={() => setShowForm(true)}
              disabled={isLoading}
              className="bg-primary hover:bg-primary/90 text-primary-foreground"
            >
              {hasInstances ? 'Add Instance' : 'Connect Gerrit'}
            </Button>
          )}
        </div>
      </div>
    </Card>
  )
}
