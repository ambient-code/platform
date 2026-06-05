'use client'

import { useState, useMemo, useEffect, useCallback } from 'react'
import { useQueries } from '@tanstack/react-query'
import { KeyRound, Plus, AlertTriangle, Settings2, Link } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/empty-state'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCredentials } from '@/queries/use-credentials'
import { useAllRoleBindings } from '@/queries/use-role-bindings'
import { useProjects } from '@/queries/use-projects'
import { useRoles } from '@/queries/use-roles'
import { queryKeys } from '@/queries/query-keys'
import { createAgentsAdapter } from '@/adapters/sdk-agents'
import type { DomainAgent, DomainCredential } from '@/domain/types'
import { CredentialTable } from './_components/credential-table'
import { CredentialCreateSheet } from './_components/credential-create-sheet'
import { CredentialManageSheet } from './_components/credential-manage-sheet'
import { BindingMatrix } from './_components/binding-matrix'

const agentsAdapter = createAgentsAdapter()

const CREDENTIAL_VIEWER_ROLE_NAME = 'credential:viewer'

export default function CredentialsPage() {
  const [createSheetOpen, setCreateSheetOpen] = useState(false)
  const [managedCredential, setManagedCredential] = useState<DomainCredential | null>(null)
  const [activeTab, setActiveTab] = useState('registry')
  const [matrixCredentialFilter, setMatrixCredentialFilter] = useState<string | undefined>(undefined)

  useEffect(() => {
    const params = new URL(window.location.href).searchParams
    const tab = params.get('tab')
    const cred = params.get('credential')
    if (tab) setActiveTab(tab)
    if (cred) {
      setMatrixCredentialFilter(cred)
      if (!tab) setActiveTab('access-matrix')
    }
  }, [])

  // Close manage sheet on browser back
  useEffect(() => {
    const handlePopState = () => {
      const manage = new URL(window.location.href).searchParams.get('manage')
      if (!manage) setManagedCredential(null)
    }
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  const openManageSheet = useCallback((cred: DomainCredential) => {
    setManagedCredential(cred)
    const url = new URL(window.location.href)
    url.searchParams.set('manage', cred.id)
    window.history.pushState({}, '', url.toString())
  }, [])

  const closeManageSheet = useCallback(() => {
    setManagedCredential(null)
    const url = new URL(window.location.href)
    if (url.searchParams.has('manage')) {
      url.searchParams.delete('manage')
      window.history.pushState({}, '', url.toString())
    }
  }, [])

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    const url = new URL(window.location.href)
    url.searchParams.set('tab', value)
    if (value !== 'access-matrix') {
      url.searchParams.delete('credential')
      setMatrixCredentialFilter(undefined)
    }
    window.history.replaceState({}, '', url.toString())
  }

  const handleNavigateToMatrix = (credentialName: string) => {
    setMatrixCredentialFilter(credentialName)
    const url = new URL(window.location.href)
    url.searchParams.set('tab', 'access-matrix')
    url.searchParams.set('credential', credentialName)
    window.history.replaceState({}, '', url.toString())
    setActiveTab('access-matrix')
  }

  const { data, isLoading, error } = useCredentials()
  const { data: projectsData } = useProjects()
  const { data: allBindings } = useAllRoleBindings("scope = 'credential'")
  const { data: rolesData, isLoading: rolesLoading } = useRoles({ size: 100 })

  const projects = useMemo(() => projectsData?.items ?? [], [projectsData])
  const bindings = useMemo(() => allBindings ?? [], [allBindings])

  const agentQueries = useQueries({
    queries: projects.map((p) => ({
      queryKey: queryKeys.agents.list(p.id, { size: 100 }),
      queryFn: () => agentsAdapter.list(p.id, { size: 100 }),
      enabled: projects.length > 0,
      staleTime: 60_000,
    })),
  })

  const allAgents = useMemo<DomainAgent[]>(() => {
    return agentQueries.flatMap((q) => q.data?.items ?? [])
  }, [agentQueries])

  const credentialViewerRoleId = useMemo(() => {
    const roles = rolesData?.items ?? []
    const role = roles.find((r) => r.name === CREDENTIAL_VIEWER_ROLE_NAME)
    return role?.id ?? null
  }, [rolesData])

  // Restore manage sheet from ?manage= URL param
  useEffect(() => {
    const manageId = new URL(window.location.href).searchParams.get('manage')
    const items = data?.items
    if (manageId && items && items.length > 0 && !managedCredential) {
      const cred = items.find((c) => c.id === manageId)
      if (cred) setManagedCredential(cred)
    }
  }, [data, managedCredential])

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold tracking-tight">Credentials</h1>
        <p className="text-sm text-destructive">
          Failed to load credentials. Please try again later.
        </p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold tracking-tight">Credentials</h1>
        <div className="space-y-3">
          <Skeleton className="h-10 w-64" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      </div>
    )
  }

  const credentials = data?.items ?? []

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">Credentials</h1>
        <Button onClick={() => setCreateSheetOpen(true)}>
          <Plus className="size-4" />
          New Credential
        </Button>
      </div>

      <Tabs value={activeTab} onValueChange={handleTabChange}>
        <TabsList className="w-full *:flex-1">
          <TabsTrigger value="registry">
            <Settings2 className="size-4 mr-1.5" />
            Manage
          </TabsTrigger>
          <TabsTrigger value="access-matrix">
            <Link className="size-4 mr-1.5" />
            Bindings
          </TabsTrigger>
        </TabsList>

        <TabsContent value="registry">
          {credentials.length === 0 ? (
            <EmptyState
              icon={KeyRound}
              title="No credentials"
              description="Add API keys, tokens, and other secrets to use with your agents."
              action={
                <Button onClick={() => setCreateSheetOpen(true)}>
                  <Plus className="size-4 mr-1.5" />
                  Add Credential
                </Button>
              }
            />
          ) : (
            <CredentialTable
              credentials={credentials}
              bindings={bindings}
              onNavigateToMatrix={handleNavigateToMatrix}
              onEditCredential={openManageSheet}
            />
          )}
        </TabsContent>

        <TabsContent value="access-matrix">
          {credentials.length === 0 || projects.length === 0 ? (
            <EmptyState
              icon={KeyRound}
              title="No data for matrix"
              description={credentials.length === 0
                ? "Add credentials to see the access matrix."
                : "Create a project to manage credential bindings."}
            />
          ) : rolesLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-10 w-64" />
              <Skeleton className="h-[300px] w-full" />
            </div>
          ) : credentialViewerRoleId === null ? (
            <div className="flex items-center gap-2 rounded-md border border-amber-500/50 bg-amber-50 dark:bg-amber-950/30 px-4 py-3 text-sm text-amber-800 dark:text-amber-200">
              <AlertTriangle className="h-4 w-4 shrink-0" />
              <span>
                The <code className="font-mono text-xs">{CREDENTIAL_VIEWER_ROLE_NAME}</code> role
                was not found. Create the role before managing credential access.
              </span>
            </div>
          ) : (
            <BindingMatrix
              credentials={credentials}
              projects={projects}
              agents={allAgents}
              bindings={bindings}
              roleId={credentialViewerRoleId}
              initialFilter={matrixCredentialFilter}
              onEditCredential={openManageSheet}
            />
          )}
        </TabsContent>
      </Tabs>

      <CredentialCreateSheet open={createSheetOpen} onOpenChange={setCreateSheetOpen} />
      {managedCredential && (
        <CredentialManageSheet
          credential={managedCredential}
          open
          onOpenChange={(open) => { if (!open) closeManageSheet() }}
          onNavigateToMatrix={handleNavigateToMatrix}
        />
      )}
    </div>
  )
}
