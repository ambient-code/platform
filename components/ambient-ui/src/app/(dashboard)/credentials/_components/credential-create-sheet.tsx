'use client'

import { useState, useMemo } from 'react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toast } from 'sonner'
import { useCreateCredential } from '@/queries/use-credentials'
import type { DomainCredentialCreateRequest } from '@/domain/types'
import {
  CREDENTIAL_CATEGORIES,
  getCategoryForProvider,
  getProviderMeta,
} from '@/domain/credential-providers'
import type { ProviderMeta } from '@/domain/credential-providers'

export function CredentialCreateSheet({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const createCredential = useCreateCredential()

  const [provider, setProvider] = useState('')
  const [name, setName] = useState('')
  const [token, setToken] = useState('')
  const [url, setUrl] = useState('')
  const [email, setEmail] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)

  const providerMeta: ProviderMeta | undefined = useMemo(
    () => (provider ? getProviderMeta(provider) : undefined),
    [provider],
  )

  const requiredFields = providerMeta?.fields ?? []

  function resetForm() {
    setProvider('')
    setName('')
    setToken('')
    setUrl('')
    setEmail('')
    setDescription('')
    setError(null)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)

    if (!name.trim()) {
      setError('Name is required.')
      return
    }

    if (!provider) {
      setError('Provider is required.')
      return
    }

    if (requiredFields.includes('token') && !token) {
      setError('Token is required for this provider.')
      return
    }

    const request: DomainCredentialCreateRequest = {
      name: name.trim(),
      provider,
    }

    if (token) request.token = token
    if (url.trim()) request.url = url.trim()
    if (email.trim()) request.email = email.trim()
    if (description.trim()) request.description = description.trim()

    try {
      await createCredential.mutateAsync(request)
      toast.success(`Credential "${name.trim()}" created`)
      resetForm()
      onOpenChange(false)
    } catch (err) {
      console.error('create credential failed', err)
      setError('Failed to create credential. Please try again.')
    }
  }

  // Auto-derive category from the selected provider
  const derivedCategory = provider ? getCategoryForProvider(provider) : undefined

  return (
    <Sheet
      open={open}
      onOpenChange={(v) => {
        if (!v) resetForm()
        onOpenChange(v)
      }}
    >
      <SheetContent side="right" className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>New Credential</SheetTitle>
          <SheetDescription>
            Add a new API key, token, or secret for use with your agents.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4 px-4 pb-4">
          <div className="space-y-1.5">
            <label htmlFor="cred-provider" className="text-sm font-medium">
              Provider <span className="text-destructive">*</span>
            </label>
            <Select
              value={provider}
              onValueChange={setProvider}
            >
              <SelectTrigger id="cred-provider" className="w-full">
                <SelectValue placeholder="Select a provider" />
              </SelectTrigger>
              <SelectContent>
                {CREDENTIAL_CATEGORIES.map((cat) => (
                  <SelectGroup key={cat.label}>
                    <SelectLabel>{cat.label}</SelectLabel>
                    {cat.providers.map((p) => (
                      <SelectItem key={p.provider} value={p.provider}>
                        {p.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                ))}
              </SelectContent>
            </Select>
            {derivedCategory && (
              <p className="text-xs text-muted-foreground">Category: {derivedCategory}</p>
            )}
          </div>

          <div className="space-y-1.5">
            <label htmlFor="cred-name" className="text-sm font-medium">
              Name <span className="text-destructive">*</span>
            </label>
            <Input
              id="cred-name"
              placeholder="my-api-key"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          {requiredFields.includes('token') && (
            <div className="space-y-1.5">
              <label htmlFor="cred-token" className="text-sm font-medium">
                Token <span className="text-destructive">*</span>
              </label>
              <Input
                id="cred-token"
                type="password"
                placeholder="Enter token or API key"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                autoComplete="off"
              />
            </div>
          )}

          {requiredFields.includes('url') && (
            <div className="space-y-1.5">
              <label htmlFor="cred-url" className="text-sm font-medium">
                URL <span className="text-destructive">*</span>
              </label>
              <Input
                id="cred-url"
                type="url"
                placeholder="https://api.example.com"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
              />
            </div>
          )}

          {requiredFields.includes('email') && (
            <div className="space-y-1.5">
              <label htmlFor="cred-email" className="text-sm font-medium">
                Email <span className="text-destructive">*</span>
              </label>
              <Input
                id="cred-email"
                type="email"
                placeholder="user@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
          )}

          <div className="space-y-1.5">
            <label htmlFor="cred-description" className="text-sm font-medium">
              Description
            </label>
            <Textarea
              id="cred-description"
              placeholder="What is this credential used for?"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="min-h-20"
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <SheetFooter className="px-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                resetForm()
                onOpenChange(false)
              }}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createCredential.isPending || !name.trim() || !provider}
            >
              {createCredential.isPending ? 'Creating...' : 'Create Credential'}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
