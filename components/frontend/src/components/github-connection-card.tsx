'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Loader2, Eye, EyeOff, ChevronDown, ChevronUp } from 'lucide-react'
import { useDisconnectGitHub, useSaveGitHubPAT, useDeleteGitHubPAT } from '@/services/queries'
import { successToast, errorToast } from '@/hooks/use-toast'

type Props = {
  appSlug?: string
  showManageButton?: boolean
  status?: {
    installed: boolean
    installationId?: number
    githubUserId?: string
    host?: string
    updatedAt?: string
    pat?: {
      configured: boolean
      updatedAt?: string
    }
    active?: string
  }
  onRefresh?: () => void
}

export function GitHubConnectionCard({ appSlug, showManageButton = true, status, onRefresh }: Props) {
  const disconnectMutation = useDisconnectGitHub()
  const savePATMutation = useSaveGitHubPAT()
  const deletePATMutation = useDeleteGitHubPAT()
  
  const [showPATSection, setShowPATSection] = useState(false)
  const [patToken, setPATToken] = useState('')
  const [showToken, setShowToken] = useState(false)
  
  const isLoading = !status
  const patStatus = status?.pat as { configured: boolean; updatedAt?: string; valid?: boolean } | undefined

  const handleConnect = () => {
    if (!appSlug) return
    const setupUrl = new URL('/integrations/github/setup', window.location.origin)
    const redirectUri = encodeURIComponent(setupUrl.toString())
    const url = `https://github.com/apps/${appSlug}/installations/new?redirect_uri=${redirectUri}`
    window.location.href = url
  }

  const handleDisconnect = async () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        successToast('GitHub disconnected successfully')
        onRefresh?.()
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to disconnect GitHub')
      },
    })
  }

  const handleManage = () => {
    window.open('https://github.com/settings/installations', '_blank')
  }

  const handleSavePAT = async () => {
    if (!patToken) {
      errorToast('Please enter a GitHub Personal Access Token')
      return
    }

    savePATMutation.mutate(patToken, {
      onSuccess: () => {
        successToast('GitHub PAT saved successfully')
        setPATToken('')
        setShowToken(false)
        onRefresh?.()
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to save GitHub PAT')
      },
    })
  }

  const handleDeletePAT = async () => {
    deletePATMutation.mutate(undefined, {
      onSuccess: () => {
        successToast('GitHub PAT removed successfully')
        onRefresh?.()
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to remove GitHub PAT')
      },
    })
  }

  return (
    <Card className="bg-card border border-gray-200 shadow-sm flex flex-col h-full">
      <div className="p-6 flex flex-col flex-1">
        {/* Header section with icon and title */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-slate-950 dark:bg-black rounded-lg flex items-center justify-center">
            <svg className="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">GitHub</h3>
            <p className="text-muted-foreground">Connect to GitHub repositories</p>
          </div>
        </div>

        {/* Status section */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span className={`w-2 h-2 rounded-full ${status?.installed || patStatus?.configured ? 'bg-green-500' : 'bg-gray-400'}`}></span>
            <span className="text-sm font-medium text-foreground/80">
              {status?.installed ? (
                <>Connected{status.githubUserId ? ` as ${status.githubUserId}` : ''}</>
              ) : patStatus?.configured ? (
                'Connected via PAT'
              ) : (
                'Not Connected'
              )}
            </span>
          </div>
          {patStatus?.configured && (
            <p className="text-xs text-muted-foreground mb-2">
              Active: Personal Access Token (overrides GitHub App)
            </p>
          )}
          {status?.installed && !patStatus?.configured && (
            <p className="text-xs text-muted-foreground mb-2">
              Active: GitHub App
            </p>
          )}
          <p className="text-muted-foreground">
            Connect to GitHub to manage repositories and create pull requests
          </p>
        </div>

        {/* GitHub App section */}
        {status?.installed && (
          <div className="mb-4 pb-4 border-b">
            <h4 className="text-sm font-semibold mb-2">GitHub App</h4>
            <p className="text-xs text-muted-foreground mb-3">
              Connected as {status.githubUserId}
            </p>
            <div className="flex gap-2">
              {showManageButton && (
                <Button 
                  variant="outline" 
                  size="sm"
                  onClick={handleManage} 
                  disabled={isLoading || disconnectMutation.isPending}
                >
                  Manage in GitHub
                </Button>
              )}
              <Button 
                variant="destructive" 
                size="sm"
                onClick={handleDisconnect} 
                disabled={isLoading || disconnectMutation.isPending}
              >
                Disconnect App
              </Button>
            </div>
          </div>
        )}

        {/* Personal Access Token section */}
        <div className="mb-4">
          <button
            onClick={() => setShowPATSection(!showPATSection)}
            className="flex items-center gap-2 text-sm font-semibold text-foreground/80 hover:text-foreground mb-3"
          >
            {showPATSection ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
            Personal Access Token {patStatus?.configured && '(Active)'}
          </button>
          
          {showPATSection && (
            <div className="space-y-3 pl-6 border-l-2 border-blue-500/20">
              <p className="text-xs text-muted-foreground">
                {patStatus?.configured 
                  ? 'PAT is configured and will be used instead of GitHub App for all operations'
                  : 'Alternative to GitHub App. If set, PAT takes precedence over GitHub App.'}
              </p>
              
              {patStatus?.configured ? (
                <div className="space-y-2">
                  {patStatus?.valid === false ? (
                    <p className="text-xs text-yellow-600 dark:text-yellow-400">
                      ⚠️ Token appears invalid or expired
                    </p>
                  ) : (
                    <p className="text-xs text-green-600 dark:text-green-400">
                      Configured (updated {patStatus.updatedAt ? new Date(patStatus.updatedAt).toLocaleDateString() : 'recently'})
                    </p>
                  )}
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleDeletePAT}
                    disabled={deletePATMutation.isPending}
                  >
                    {deletePATMutation.isPending ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Removing...
                      </>
                    ) : (
                      'Remove PAT'
                    )}
                  </Button>
                </div>
              ) : (
                <div className="space-y-3">
                  <div className="space-y-1">
                    <Label htmlFor="github-pat" className="text-xs">Token</Label>
                    <div className="flex gap-2">
                      <Input
                        id="github-pat"
                        type={showToken ? 'text' : 'password'}
                        placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                        value={patToken}
                        onChange={(e) => setPATToken(e.target.value)}
                        disabled={savePATMutation.isPending}
                        className="text-sm"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowToken(!showToken)}
                        disabled={savePATMutation.isPending}
                      >
                        {showToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </Button>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      Create a token with <code>repo</code> scope at{' '}
                      <a
                        href="https://github.com/settings/tokens/new"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="underline"
                      >
                        GitHub Settings
                      </a>
                    </p>
                  </div>
                  <Button
                    onClick={handleSavePAT}
                    disabled={savePATMutation.isPending || !patToken}
                    size="sm"
                  >
                    {savePATMutation.isPending ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Saving...
                      </>
                    ) : (
                      'Save PAT'
                    )}
                  </Button>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Action buttons */}
        <div className="flex gap-3 mt-auto">
          {!status?.installed && !patStatus?.configured && (
            <Button 
              onClick={handleConnect} 
              disabled={isLoading || !appSlug}
              className="bg-blue-600 hover:bg-blue-700 text-white"
            >
              Connect GitHub App
            </Button>
          )}
        </div>
      </div>
    </Card>
  )
}

