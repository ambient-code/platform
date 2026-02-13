'use client'

import React from 'react'
import { GitHubConnectionCard } from '@/components/github-connection-card'
import { GoogleDriveConnectionCard } from '@/components/google-drive-connection-card'
import { GitLabConnectionCard } from '@/components/gitlab-connection-card'
import { JiraConnectionCard } from '@/components/jira-connection-card'
import { PageHeader } from '@/components/page-header'
import { useIntegrationsStatus } from '@/services/queries/use-integrations'
import { Loader2 } from 'lucide-react'

type Props = { appSlug?: string }

export default function IntegrationsClient({ appSlug }: Props) {
  // Fetch all integration statuses in one call
  const { data: integrations, isLoading, refetch } = useIntegrationsStatus()

  return (
    <div className="min-h-screen bg-background">
      {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 border-b">
        <div className="container mx-auto px-6 py-6">
          <PageHeader
            title="Integrations"
            description="Connect Ambient Code Platform with your favorite tools and services. All integrations work across all your workspaces."
          />
        </div>
      </div>

      <div className="container mx-auto p-0">
        {/* Content */}
        <div className="px-6 pt-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <GitHubConnectionCard 
                appSlug={appSlug} 
                showManageButton={true}
                status={integrations?.github}
                onRefresh={refetch}
              />
              <GoogleDriveConnectionCard 
                showManageButton={true}
                status={integrations?.google}
                onRefresh={refetch}
              />
              <GitLabConnectionCard 
                status={integrations?.gitlab}
                onRefresh={refetch}
              />
              <JiraConnectionCard 
                status={integrations?.jira}
                onRefresh={refetch}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}


