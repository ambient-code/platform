'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Badge } from '@/components/ui/badge'
import { Loader2, Eye, EyeOff, Plus, Trash2, CheckCircle2, XCircle, Server } from 'lucide-react'
import { toast } from 'sonner'
import {
  useGerritInstances,
  useConnectGerrit,
  useDisconnectGerrit,
  useTestGerritConnection,
} from '@/services/queries/use-gerrit'
import type { GerritAuthMethod, GerritInstanceStatus } from '@/services/api/gerrit-auth'

type Props = {
  onRefresh?: () => void
}

export function GerritConnectionCard({ onRefresh }: Props) {
  const { data: instancesData, refetch: refetchInstances, isLoading, isError } = useGerritInstances()
  const connectMutation = useConnectGerrit()
  const disconnectMutation = useDisconnectGerrit()
  const testMutation = useTestGerritConnection()

  const [showForm, setShowForm] = useState(false)
  const [instanceName, setInstanceName] = useState('')
  const [url, setUrl] = useState('')
  const [authMethod, setAuthMethod] = useState<GerritAuthMethod>('http_basic')
  const [username, setUsername] = useState('')
  const [httpToken, setHttpToken] = useState('')
  const [showToken, setShowToken] = useState(false)
  const [gitcookiesContent, setGitcookiesContent] = useState('')

  const instances = instancesData?.instances ?? []
  const hasInstances = instances.length > 0

  const resetForm = () => {
    setInstanceName('')
    setUrl('')
    setAuthMethod('http_basic')
    setUsername('')
    setHttpToken('')
    setShowToken(false)
    setGitcookiesContent('')
  }

  const handleAuthMethodChange = (value: string) => {
    const method = value as GerritAuthMethod
    setAuthMethod(method)
    if (method === 'http_basic') {
      setGitcookiesContent('')
    } else {
      setUsername('')
      setHttpToken('')
    }
  }

  const normalizedInstanceName = instanceName.toLowerCase().replace(/[^a-z0-9-]/g, '-')

  const isFormValid = () => {
    if (!normalizedInstanceName || !url) return false
    if (authMethod === 'http_basic') return !!username && !!httpToken
    return !!gitcookiesContent
  }

  const buildTestPayload = () => {
    if (authMethod === 'http_basic') {
      return { url, authMethod: 'http_basic' as const, username, httpToken }
    }
    return { url, authMethod: 'git_cookies' as const, gitcookiesContent }
  }

  const buildConnectPayload = () => {
    if (authMethod === 'http_basic') {
      return { instanceName: normalizedInstanceName, url, authMethod: 'http_basic' as const, username, httpToken }
    }
    return { instanceName: normalizedInstanceName, url, authMethod: 'git_cookies' as const, gitcookiesContent }
  }

  const handleTest = () => {
    if (!url) {
      toast.error('Please enter a Gerrit URL')
      return
    }
    if (authMethod === 'http_basic' && (!username || !httpToken)) {
      toast.error('Please enter username and HTTP token')
      return
    }
    if (authMethod === 'git_cookies' && !gitcookiesContent) {
      toast.error('Please enter gitcookies content')
      return
    }

    testMutation.mutate(buildTestPayload(), {
      onSuccess: (result) => {
        if (result.valid) {
          toast.success(result.message ?? 'Connection test successful')
        } else {
          toast.error(result.error ?? 'Connection test failed')
        }
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Connection test failed')
      },
    })
  }

  const handleConnect = () => {
    if (!isFormValid()) {
      toast.error('Please fill in all required fields')
      return
    }

    connectMutation.mutate(buildConnectPayload(), {
      onSuccess: () => {
        toast.success(`Gerrit instance "${normalizedInstanceName}" connected`)
        setShowForm(false)
        resetForm()
        onRefresh?.()
        refetchInstances()
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Failed to connect Gerrit instance')
      },
    })
  }

  const handleDisconnect = (name: string) => {
    disconnectMutation.mutate(name, {
      onSuccess: () => {
        toast.success(`Gerrit instance "${name}" disconnected`)
        onRefresh?.()
        refetchInstances()
      },
      onError: (error) => {
        toast.error(error instanceof Error ? error.message : 'Failed to disconnect Gerrit instance')
      },
    })
  }

  return (
    <Card className="bg-card border border-border/60 shadow-sm shadow-black/[0.03] dark:shadow-black/[0.15] flex flex-col h-full">
      <div className="p-6 flex flex-col flex-1">
        {/* Header */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-primary rounded-lg flex items-center justify-center">
            <Server className="w-10 h-10 text-white" aria-hidden="true" />
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">Gerrit</h3>
            <p className="text-muted-foreground">Connect to Gerrit for code review</p>
          </div>
        </div>

        {/* Loading / error states */}
        {isLoading && (
          <div className="mb-4 flex items-center gap-2 text-muted-foreground">
            <Loader2 className="w-4 h-4 animate-spin" />
            <span className="text-sm">Loading instances…</span>
          </div>
        )}
        {isError && !isLoading && (
          <div className="mb-4 space-y-2">
            <p className="text-sm text-destructive">Failed to load Gerrit instances.</p>
            <Button variant="outline" size="sm" onClick={() => refetchInstances()}>Retry</Button>
          </div>
        )}

        {/* Instance list */}
        {!isLoading && !isError && hasInstances && (
          <div className="mb-4 space-y-2">
            {instances.map((instance: GerritInstanceStatus) => (
              <div
                key={instance.instanceName}
                className="flex items-center justify-between rounded-md border border-border/60 px-3 py-2"
              >
                <div className="flex items-center gap-2 min-w-0">
                  {instance.connected ? (
                    <CheckCircle2 className="w-4 h-4 text-green-500 flex-shrink-0" />
                  ) : (
                    <XCircle className="w-4 h-4 text-red-500 flex-shrink-0" />
                  )}
                  <div className="min-w-0">
                    <span className="text-sm font-medium text-foreground truncate block">
                      {instance.instanceName}
                    </span>
                    <span className="text-xs text-muted-foreground truncate block">
                      {instance.url}
                    </span>
                  </div>
                  <Badge variant="secondary" className="ml-2 text-xs flex-shrink-0">
                    {instance.authMethod === 'http_basic' ? 'HTTP' : 'Cookies'}
                  </Badge>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDisconnect(instance.instanceName)}
                  disabled={disconnectMutation.isPending}
                  className="flex-shrink-0 text-destructive hover:text-destructive"
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            ))}
          </div>
        )}

        {/* Status when no instances */}
        {!isLoading && !isError && !hasInstances && !showForm && (
          <div className="mb-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="w-2 h-2 rounded-full bg-gray-400" />
              <span className="text-sm font-medium text-foreground/80">Not Connected</span>
            </div>
            <p className="text-muted-foreground">
              Connect to Gerrit instances for code review across all sessions
            </p>
          </div>
        )}

        {/* Add instance form */}
        {showForm && (
          <div className="mb-4 space-y-3">
            <div>
              <Label htmlFor="gerrit-instance-name" className="text-sm">Instance Name</Label>
              <Input
                id="gerrit-instance-name"
                type="text"
                placeholder="e.g. my-gerrit"
                value={instanceName}
                onChange={(e) => setInstanceName(e.target.value)}
                disabled={connectMutation.isPending}
                className="mt-1"
              />
              {instanceName && instanceName !== normalizedInstanceName && (
                <p className="text-xs text-muted-foreground mt-1">
                  Will be saved as: {normalizedInstanceName}
                </p>
              )}
            </div>
            <div>
              <Label htmlFor="gerrit-url" className="text-sm">Gerrit URL</Label>
              <Input
                id="gerrit-url"
                type="url"
                placeholder="https://review.example.com"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                disabled={connectMutation.isPending}
                className="mt-1"
              />
            </div>
            <div>
              <Label className="text-sm">Authentication Method</Label>
              <RadioGroup
                value={authMethod}
                onValueChange={handleAuthMethodChange}
                className="mt-2"
              >
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="http_basic" id="auth-http" />
                  <Label htmlFor="auth-http" className="text-sm font-normal cursor-pointer">
                    HTTP Basic (Username + Token)
                  </Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="git_cookies" id="auth-cookies" />
                  <Label htmlFor="auth-cookies" className="text-sm font-normal cursor-pointer">
                    Git Cookies (.gitcookies)
                  </Label>
                </div>
              </RadioGroup>
            </div>

            {authMethod === 'http_basic' && (
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
                  <Label htmlFor="gerrit-token" className="text-sm">HTTP Token</Label>
                  <div className="flex gap-2 mt-1">
                    <Input
                      id="gerrit-token"
                      type={showToken ? 'text' : 'password'}
                      placeholder="Your Gerrit HTTP token"
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
                    Generate at Settings &rarr; HTTP Credentials in your Gerrit instance
                  </p>
                </div>
              </>
            )}

            {authMethod === 'git_cookies' && (
              <div>
                <Label htmlFor="gerrit-gitcookies" className="text-sm">Git Cookies Content</Label>
                <Textarea
                  id="gerrit-gitcookies"
                  placeholder="Paste your .gitcookies file content"
                  value={gitcookiesContent}
                  onChange={(e) => setGitcookiesContent(e.target.value)}
                  disabled={connectMutation.isPending}
                  className="mt-1 font-mono text-xs"
                  rows={4}
                />
                <p className="text-xs text-muted-foreground mt-1">
                  Contents of your ~/.gitcookies file for this Gerrit instance
                </p>
              </div>
            )}

            <div className="flex gap-2 pt-2">
              <Button
                variant="outline"
                onClick={handleTest}
                disabled={!url || testMutation.isPending || connectMutation.isPending}
              >
                {testMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Testing...
                  </>
                ) : (
                  'Test Connection'
                )}
              </Button>
              <Button
                onClick={handleConnect}
                disabled={!isFormValid() || connectMutation.isPending}
                className="flex-1"
              >
                {connectMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Saving...
                  </>
                ) : (
                  'Save Instance'
                )}
              </Button>
              <Button
                variant="outline"
                onClick={() => {
                  setShowForm(false)
                  resetForm()
                }}
                disabled={connectMutation.isPending}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        {/* Add instance button */}
        {!showForm && (
          <div className="flex gap-3 mt-auto">
            <Button
              onClick={() => setShowForm(true)}
              className="bg-primary hover:bg-primary/90 text-primary-foreground"
            >
              <Plus className="w-4 h-4 mr-2" />
              Add Instance
            </Button>
          </div>
        )}
      </div>
    </Card>
  )
}
