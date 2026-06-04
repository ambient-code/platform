'use client'

import { useState, useMemo } from 'react'
import { KeyRound, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/empty-state'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCredentials } from '@/queries/use-credentials'
import { useRoleBindings } from '@/queries/use-role-bindings'
import { useProjects } from '@/queries/use-projects'
import { CredentialTable } from './_components/credential-table'
import { CredentialCreateSheet } from './_components/credential-create-sheet'
import { BindingMatrix } from './_components/binding-matrix'

export default function CredentialsPage() {
  const [createSheetOpen, setCreateSheetOpen] = useState(false)
  const { data, isLoading, error } = useCredentials()
  const { data: projectsData } = useProjects()
  const { data: bindingsData } = useRoleBindings({ size: 1000, search: "scope = 'credential'" })

  const projects = useMemo(() => projectsData?.items ?? [], [projectsData])
  const bindings = useMemo(() => bindingsData?.items ?? [], [bindingsData])

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-semibold">Credentials</h1>
        <p className="text-sm text-destructive">
          Failed to load credentials. Please try again later.
        </p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-semibold">Credentials</h1>
        <div className="space-y-3">
          <Skeleton className="h-10 w-64" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      </div>
    )
  }

  const credentials = data?.items ?? []

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Credentials</h1>
        <Button size="sm" onClick={() => setCreateSheetOpen(true)}>
          <Plus className="size-4" />
          New Credential
        </Button>
      </div>

      <Tabs defaultValue="registry">
        <TabsList>
          <TabsTrigger value="registry">Registry</TabsTrigger>
          <TabsTrigger value="access-matrix">Access Matrix</TabsTrigger>
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
            <CredentialTable credentials={credentials} />
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
          ) : (
            <BindingMatrix
              credentials={credentials}
              projects={projects}
              agents={[]}
              bindings={bindings}
              roleId=""
            />
          )}
        </TabsContent>
      </Tabs>

      <CredentialCreateSheet open={createSheetOpen} onOpenChange={setCreateSheetOpen} />
    </div>
  )
}
