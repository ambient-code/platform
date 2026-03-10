'use client'

import type { ReactNode } from 'react'
import { GitHubConnectionCard } from '@/components/github-connection-card'
import { GoogleDriveConnectionCard } from '@/components/google-drive-connection-card'
import { GitLabConnectionCard } from '@/components/gitlab-connection-card'
import { JiraConnectionCard } from '@/components/jira-connection-card'
import { MCPCredentialCard } from '@/components/mcp-credential-card'
import { PageHeader } from '@/components/page-header'
import { useIntegrationsStatus } from '@/services/queries/use-integrations'
import { Loader2 } from 'lucide-react'

type FieldDefinition = {
  name: string
  label: string
  type: 'text' | 'password'
  placeholder?: string
  helpText?: string
}

type MCPCredentialServer = {
  displayName: string
  description: string
  iconBg?: string
  icon?: ReactNode
  fields: FieldDefinition[]
}

/**
 * MCP servers that require user-provided credentials.
 * Platform-wide no-auth servers (context7, deepwiki, webfetch) are not shown
 * here — they belong at the project scope, not user-scoped integrations.
 */
const MCP_CREDENTIAL_SERVERS: Record<string, MCPCredentialServer> = {
  'mcp-atlassian': {
    displayName: 'Jira (MCP)',
    description:
      'Provide Jira credentials for the MCP Atlassian server used in agentic sessions',
    iconBg: 'bg-blue-600',
    icon: (
      <svg className="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M11.571 11.513H0a5.218 5.218 0 0 0 5.232 5.215h2.13v2.057A5.215 5.215 0 0 0 12.575 24V12.518a1.005 1.005 0 0 0-1.005-1.005zm5.723-5.756H5.736a5.215 5.215 0 0 0 5.215 5.214h2.129v2.058a5.218 5.218 0 0 0 5.215 5.214V6.758a1.001 1.001 0 0 0-1.001-1.001zM23.013 0H11.455a5.215 5.215 0 0 0 5.215 5.215h2.129v2.057A5.215 5.215 0 0 0 24 12.483V1.005A1.001 1.001 0 0 0 23.013 0z" />
      </svg>
    ),
    fields: [
      {
        name: 'jira_url',
        label: 'Jira URL',
        type: 'text',
        placeholder: 'https://issues.redhat.com',
      },
      {
        name: 'jira_email',
        label: 'Email',
        type: 'text',
        placeholder: 'you@example.com',
      },
      {
        name: 'jira_api_token',
        label: 'API Token',
        type: 'password',
        placeholder: 'Your Jira API token',
        helpText: 'Create a token in Jira > Profile > Personal Access Tokens',
      },
    ],
  },
}

type Props = { appSlug?: string }

export default function IntegrationsClient({ appSlug }: Props) {
  const { data: integrations, isLoading, refetch } = useIntegrationsStatus()

  // Merge known credential servers with any additional servers that have stored credentials
  const mcpServerNames = new Set<string>(Object.keys(MCP_CREDENTIAL_SERVERS))
  if (integrations?.mcpServers) {
    for (const name of Object.keys(integrations.mcpServers)) {
      mcpServerNames.add(name)
    }
  }

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
            <>
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

              {/* MCP Servers section — only credential-based servers (user-scoped) */}
              {mcpServerNames.size > 0 && (
                <div className="mt-8">
                  <h2 className="text-lg font-semibold text-foreground mb-4">MCP Servers</h2>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {[...mcpServerNames].map((name) => {
                      const entry = MCP_CREDENTIAL_SERVERS[name]
                      const serverStatus = integrations?.mcpServers?.[name]
                      return (
                        <MCPCredentialCard
                          key={name}
                          serverName={name}
                          displayName={entry?.displayName ?? name}
                          icon={entry?.icon}
                          iconBg={entry?.iconBg}
                          description={
                            entry?.description ?? `Credentials for the ${name} MCP server`
                          }
                          fields={
                            entry?.fields ?? [
                              {
                                name: 'api_key',
                                label: 'API Key',
                                type: 'password' as const,
                                placeholder: 'Enter API key',
                              },
                            ]
                          }
                          status={
                            serverStatus
                              ? { connected: serverStatus.connected, serverName: name }
                              : { connected: false, serverName: name }
                          }
                          onRefresh={refetch}
                        />
                      )
                    })}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
