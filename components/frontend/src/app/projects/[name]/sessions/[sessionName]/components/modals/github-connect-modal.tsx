'use client'

import { useEffect, useState } from 'react'
import { Eye, EyeOff, Loader2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useIntegrationSecrets, useUpdateIntegrationSecrets } from '@/services/queries/use-secrets'
import { useQueryClient } from '@tanstack/react-query'
import { projectKeys } from '@/services/queries/use-projects'
import { successToast, errorToast } from '@/hooks/use-toast'

type GitHubConnectModalProps = {
  projectName: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function GitHubConnectModal({
  projectName,
  open,
  onOpenChange,
}: GitHubConnectModalProps) {
  const queryClient = useQueryClient()
  const { data: integrationSecrets } = useIntegrationSecrets(projectName)
  const updateMutation = useUpdateIntegrationSecrets()

  const [gitUserName, setGitUserName] = useState('')
  const [gitUserEmail, setGitUserEmail] = useState('')
  const [gitToken, setGitToken] = useState('')
  const [showGitToken, setShowGitToken] = useState(false)

  // Sync form from integration secrets when modal opens or data loads
  useEffect(() => {
    if (!open || !integrationSecrets) return
    const byKey: Record<string, string> = Object.fromEntries(
      integrationSecrets.map((s) => [s.key, s.value])
    )
    setGitUserName(byKey['GIT_USER_NAME'] ?? '')
    setGitUserEmail(byKey['GIT_USER_EMAIL'] ?? '')
    setGitToken(byKey['GITHUB_TOKEN'] ?? '')
  }, [open, integrationSecrets])

  const handleSave = () => {
    if (!projectName) return

    // Merge current integration secrets with GitHub fields so we don't overwrite Jira, etc.
    const current: Record<string, string> = integrationSecrets
      ? Object.fromEntries(integrationSecrets.map((s) => [s.key, s.value]))
      : {}
    const merged: Record<string, string> = {
      ...current,
      GIT_USER_NAME: gitUserName.trim(),
      GIT_USER_EMAIL: gitUserEmail.trim(),
      GITHUB_TOKEN: gitToken,
    }

    const secrets = Object.entries(merged).map(([key, value]) => ({ key, value }))
    updateMutation.mutate(
      { projectName, secrets },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: projectKeys.integrationStatus(projectName) })
          successToast('GitHub connection saved')
          onOpenChange(false)
        },
        onError: (err) => {
          errorToast(err instanceof Error ? err.message : 'Failed to save GitHub connection')
        },
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Connect GitHub</DialogTitle>
          <DialogDescription>
            Configure Git credentials for repository operations (clone, commit, push). These are
            stored in workspace settings.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label htmlFor="github-connect-git-user-name">GIT_USER_NAME</Label>
              <Input
                id="github-connect-git-user-name"
                placeholder="Your Name"
                value={gitUserName}
                onChange={(e) => setGitUserName(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="github-connect-git-user-email">GIT_USER_EMAIL</Label>
              <Input
                id="github-connect-git-user-email"
                placeholder="you@example.com"
                value={gitUserEmail}
                onChange={(e) => setGitUserEmail(e.target.value)}
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="github-connect-git-token">GITHUB_TOKEN</Label>
            <p className="text-xs text-muted-foreground">
              GitHub personal access token or fine-grained token for git operations and API access
            </p>
            <div className="flex items-center gap-2">
              <Input
                id="github-connect-git-token"
                type={showGitToken ? 'text' : 'password'}
                placeholder="ghp_... or glpat-..."
                value={gitToken}
                onChange={(e) => setGitToken(e.target.value)}
                className="flex-1"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setShowGitToken((v) => !v)}
                aria-label={showGitToken ? 'Hide token' : 'Show token'}
              >
                {showGitToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </Button>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={updateMutation.isPending}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={updateMutation.isPending}>
            {updateMutation.isPending ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : (
              'Save'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
