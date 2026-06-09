'use client'

import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useCreateProject } from '@/queries/use-projects'

const DNS_1123_REGEX = /^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z][a-z0-9]?$/

function validateProjectName(name: string): string | null {
  if (name.length < 2) return 'Name must be at least 2 characters'
  if (name.length > 63) return 'Name must be 63 characters or fewer'
  if (!DNS_1123_REGEX.test(name))
    return 'Lowercase letters, numbers, and hyphens only. Must start with a letter and end with a letter or number.'
  return null
}

export function CreateProjectDialog() {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [nameError, setNameError] = useState<string | null>(null)
  const createProject = useCreateProject()

  function resetForm() {
    setName('')
    setDescription('')
    setNameError(null)
    createProject.reset()
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const error = validateProjectName(name)
    if (error) {
      setNameError(error)
      return
    }
    setNameError(null)
    createProject.mutate(
      { name, description: description || undefined },
      {
        onSuccess: () => {
          setOpen(false)
          resetForm()
        },
      },
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        setOpen(v)
        if (!v) resetForm()
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="mr-1.5 h-4 w-4" />
          New Project
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Project</DialogTitle>
            <DialogDescription>
              Projects group your agents, sessions, and credentials.
            </DialogDescription>
          </DialogHeader>
          <div className="mt-4 space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium" htmlFor="project-name">Name</label>
              <Input
                id="project-name"
                value={name}
                onChange={(e) => {
                  setName(e.target.value.toLowerCase())
                  setNameError(null)
                }}
                placeholder="my-project"
                aria-describedby={nameError ? 'project-name-error' : undefined}
                aria-invalid={!!nameError}
                autoFocus
              />
              {nameError && (
                <p id="project-name-error" className="text-sm text-destructive">
                  {nameError}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium" htmlFor="project-description">
                Description <span className="text-muted-foreground">(optional)</span>
              </label>
              <Textarea
                id="project-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="What is this project for?"
                rows={2}
              />
            </div>
            {createProject.isError && (
              <p className="text-sm text-destructive">
                Failed to create project. Please try again.
              </p>
            )}
          </div>
          <DialogFooter className="mt-6">
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createProject.isPending}>
              {createProject.isPending ? 'Creating...' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
