'use client'

import { useRouter } from 'next/navigation'
import { useProjects } from '@/queries/use-projects'
import { domainProbe } from '@/lib/observability'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'

type ProjectSelectorProps = {
  projectId: string | null
}

export function ProjectSelector({ projectId }: ProjectSelectorProps) {
  const router = useRouter()
  const { data, isLoading } = useProjects()

  if (isLoading) {
    return <Skeleton className="h-9 w-full" />
  }

  const projects = data?.items ?? []

  return (
    <Select
      value={projectId ?? undefined}
      onValueChange={(value) => {
        domainProbe.projectSelected({ projectId: value })
        router.push(`/${value}/sessions`)
      }}
    >
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Select project..." />
      </SelectTrigger>
      <SelectContent>
        {projects.map((project) => (
          <SelectItem key={project.id} value={project.id}>
            {project.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
