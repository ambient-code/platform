'use client'

import React, { useState, useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Loader2 } from 'lucide-react'
import { successToast, errorToast } from '@/hooks/use-toast'
import { useDisconnectGoogle } from '@/services/queries/use-google'
import * as googleAuthApi from '@/services/api/google-auth'

type Props = {
  showManageButton?: boolean
  status?: {
    connected: boolean
    email?: string
    expiresAt?: string
    updatedAt?: string
  }
  onRefresh?: () => void
}

export function GoogleDriveConnectionCard({ showManageButton = true, status, onRefresh }: Props) {
  const disconnectMutation = useDisconnectGoogle()
  const isLoading = !status
  const error = null
  const [connecting, setConnecting] = useState(false)
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Cleanup polling interval on unmount
  useEffect(() => {
    return () => {
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current)
      }
    }
  }, [])

  const handleConnect = async () => {
    setConnecting(true)

    try {
      // Get OAuth URL from backend
      const response = await googleAuthApi.getGoogleOAuthURL()
      const authUrl = response.url

      // Open OAuth flow in popup window
      const width = 600
      const height = 700
      const left = window.screen.width / 2 - width / 2
      const top = window.screen.height / 2 - height / 2

      const popup = window.open(
        authUrl,
        'Google OAuth',
        `width=${width},height=${height},left=${left},top=${top}`
      )

      // Check if popup was blocked
      if (!popup || popup.closed) {
        errorToast('Popup was blocked. Please allow popups for this site and try again.')
        setConnecting(false)
        return
      }

      // Clear any existing poll timer
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current)
      }

      // Poll for popup close and check status
      pollTimerRef.current = setInterval(() => {
        if (popup.closed) {
          if (pollTimerRef.current) {
            clearInterval(pollTimerRef.current)
            pollTimerRef.current = null
          }
          setConnecting(false)
          // Refetch status to check if OAuth succeeded
          onRefresh?.()
        }
      }, 500)
    } catch (err) {
      console.error('Failed to initiate Google OAuth:', err)
      errorToast(err instanceof Error ? err.message : 'Failed to connect Google Drive')
      setConnecting(false)
    }
  }

  const handleDisconnect = async () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        successToast('Google Drive disconnected successfully')
        onRefresh?.()
      },
      onError: (error) => {
        errorToast(error instanceof Error ? error.message : 'Failed to disconnect Google Drive')
      },
    })
  }

  const handleManage = () => {
    window.open('https://myaccount.google.com/permissions', '_blank')
  }

  return (
    <Card className="bg-card border border-gray-200 shadow-sm flex flex-col h-full">
      <div className="p-6 flex flex-col flex-1">
        {/* Header section with icon and title */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-white border border-gray-200 rounded-lg flex items-center justify-center">
            <svg className="w-10 h-10" viewBox="0 0 24 24" aria-hidden="true">
              <path
                fill="#4285F4"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="#34A853"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="#FBBC05"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
              />
              <path
                fill="#EA4335"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">Google Drive</h3>
            <p className="text-muted-foreground">Access Drive files across all sessions</p>
          </div>
        </div>

        {/* Status section */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span className={`w-2 h-2 rounded-full ${error ? 'bg-red-500' : status?.connected ? 'bg-green-500' : 'bg-gray-400'}`}></span>
            <span className="text-sm font-medium text-foreground/80">
              {error ? (
                'Connection Error'
              ) : status?.connected ? (
                <>Connected{status.email ? ` as ${status.email}` : ''}</>
              ) : (
                'Not Connected'
              )}
            </span>
          </div>
          <p className="text-muted-foreground">
            {error 
              ? 'Failed to check connection status. Please try again.'
              : 'Connect to Google Drive to access files in all your sessions via MCP'
            }
          </p>
        </div>

        {/* Action buttons */}
        <div className="flex gap-3 mt-auto">
          {status?.connected ? (
            <>
              {showManageButton && (
                <Button 
                  variant="outline" 
                  onClick={handleManage} 
                  disabled={isLoading || disconnectMutation.isPending}
                >
                  Manage Permissions
                </Button>
              )}
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
          ) : (
            <Button 
              onClick={handleConnect} 
              disabled={isLoading || connecting}
              className="bg-blue-600 hover:bg-blue-700 text-white"
            >
              {connecting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Connecting...
                </>
              ) : (
                'Connect Google Drive'
              )}
            </Button>
          )}
        </div>
      </div>
    </Card>
  )
}

