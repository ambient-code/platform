'use client'

import { useRouter } from 'next/navigation'
import { FolderOpen } from 'lucide-react'
import { useProjects } from '@/queries/use-projects'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/empty-state'

function ProjectCardSkeleton() {
  return (
    <Card>
      <CardHeader>
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-4 w-48" />
      </CardHeader>
      <CardContent>
        <Skeleton className="h-4 w-24" />
      </CardContent>
    </Card>
  )
}

export default function ProjectPickerPage() {
  const router = useRouter()
  const { data, isLoading, isError } = useProjects()

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-semibold">Projects</h1>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }, (_, i) => (
            <ProjectCardSkeleton key={i} />
          ))}
        </div>
      </div>
    )
  }

  if (isError) {
    return (
      <EmptyState
        icon={FolderOpen}
        title="Failed to load projects"
        description="Something went wrong while loading your projects. Please try again."
      />
    )
  }

  const projects = data?.items ?? []

  if (projects.length === 0) {
    return (
      <EmptyState
        icon={FolderOpen}
        title="No projects found"
        description="Create a project to get started with the Ambient Code Platform."
      />
    )
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Projects</h1>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {projects.map((project) => (
          <Card
            key={project.id}
            className="cursor-pointer transition-shadow hover:shadow-md"
            onClick={() => router.push(`/${project.id}/sessions`)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                router.push(`/${project.id}/sessions`)
              }
            }}
          >
            <CardHeader>
              <CardTitle>{project.name}</CardTitle>
              {project.description && (
                <CardDescription>{project.description}</CardDescription>
              )}
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">
                {project.status ?? 'Active'}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
