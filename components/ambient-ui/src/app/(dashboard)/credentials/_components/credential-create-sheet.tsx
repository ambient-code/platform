'use client'

import { useState, useMemo, useCallback, useRef } from 'react'
import { Eye, EyeOff, Upload, X } from 'lucide-react'
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
  getProviderMeta,
} from '@/domain/credential-providers'
import type { ProviderMeta } from '@/domain/credential-providers'

type FieldErrors = {
  name?: string
  provider?: string
  token?: string
  url?: string
  email?: string
}

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
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [showSecret, setShowSecret] = useState(false)
  const [uploadedFileName, setUploadedFileName] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const providerMeta: ProviderMeta | undefined = useMemo(
    () => (provider ? getProviderMeta(provider) : undefined),
    [provider],
  )

  const fields = providerMeta?.fields ?? []
  const isUrlOptional = providerMeta?.urlOptional === true

  function resetForm() {
    setProvider('')
    setName('')
    setToken('')
    setUrl('')
    setEmail('')
    setDescription('')
    setFieldErrors({})
    setSubmitError(null)
    setShowSecret(false)
    setUploadedFileName(null)
  }

  function validate(): FieldErrors {
    const errors: FieldErrors = {}
    if (!name.trim()) errors.name = 'Name is required.'
    if (!provider) errors.provider = 'Select a provider.'
    if (fields.includes('token') && !token.trim()) {
      errors.token = `${providerMeta?.tokenField?.label ?? 'Token'} is required.`
    }
    if (fields.includes('url') && !isUrlOptional && !url.trim()) {
      errors.url = 'URL is required for this provider.'
    }
    if (fields.includes('email') && !email.trim()) {
      errors.email = 'Email is required for this provider.'
    }
    return errors
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitError(null)

    const errors = validate()
    setFieldErrors(errors)
    if (Object.keys(errors).length > 0) return

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
    } catch {
      setSubmitError('Failed to create credential. Please try again.')
    }
  }

  const MAX_UPLOAD_BYTES = 1_048_576

  const handleFileUpload = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    if (file.size > MAX_UPLOAD_BYTES) {
      setFieldErrors((prev) => ({ ...prev, token: 'File exceeds 1 MB limit.' }))
      return
    }
    const reader = new FileReader()
    reader.onload = () => {
      const text = reader.result
      if (typeof text === 'string') {
        setToken(text)
        setUploadedFileName(file.name)
        setFieldErrors((prev) => ({ ...prev, token: undefined }))
      }
    }
    reader.onerror = () => {
      setFieldErrors((prev) => ({ ...prev, token: 'Failed to read file.' }))
      setToken('')
      setUploadedFileName(null)
    }
    reader.readAsText(file)
  }, [])

  const clearUpload = useCallback(() => {
    setToken('')
    setUploadedFileName(null)
  }, [])

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
          {/* Provider */}
          <div className="space-y-1.5">
            <label htmlFor="cred-provider" className="text-sm font-medium">
              Provider <span className="text-destructive">*</span>
            </label>
            <Select
              value={provider}
              onValueChange={(v) => {
                setProvider(v)
                setToken('')
                setUrl('')
                setEmail('')
                setShowSecret(false)
                setUploadedFileName(null)
                setFieldErrors({})
              }}
            >
              <SelectTrigger
                id="cred-provider"
                className="w-full"
                aria-describedby={fieldErrors.provider ? 'err-provider' : undefined}
              >
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
            {fieldErrors.provider && (
              <p id="err-provider" className="text-xs text-destructive">{fieldErrors.provider}</p>
            )}
          </div>

          {/* Name */}
          <div className="space-y-1.5">
            <label htmlFor="cred-name" className="text-sm font-medium">
              Name <span className="text-destructive">*</span>
            </label>
            <Input
              id="cred-name"
              placeholder={providerMeta?.namePlaceholder ?? 'my-credential'}
              value={name}
              onChange={(e) => {
                setName(e.target.value)
                if (fieldErrors.name) setFieldErrors((prev) => ({ ...prev, name: undefined }))
              }}
              aria-describedby={fieldErrors.name ? 'err-name' : undefined}
            />
            {fieldErrors.name && (
              <p id="err-name" className="text-xs text-destructive">{fieldErrors.name}</p>
            )}
          </div>

          {/* Token / Secret / Kubeconfig */}
          {fields.includes('token') && (
            <div className="space-y-1.5">
              <label htmlFor="cred-token" className="text-sm font-medium">
                {providerMeta?.tokenField?.label ?? 'Token'} <span className="text-destructive">*</span>
              </label>
              {providerMeta?.tokenField?.multiline ? (
                <>
                  {uploadedFileName ? (
                    <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-2 text-sm">
                      <Upload className="size-4 text-muted-foreground shrink-0" />
                      <span className="truncate flex-1 font-mono text-xs">{uploadedFileName}</span>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-6 shrink-0"
                        onClick={clearUpload}
                        aria-label="Remove uploaded file"
                      >
                        <X className="size-3.5" />
                      </Button>
                    </div>
                  ) : (
                    <div className="relative">
                      <Textarea
                        id="cred-token"
                        placeholder={providerMeta.tokenField.placeholder}
                        value={token}
                        onChange={(e) => {
                          setToken(e.target.value)
                          if (fieldErrors.token) setFieldErrors((prev) => ({ ...prev, token: undefined }))
                        }}
                        autoComplete="off"
                        className={`min-h-32 font-mono text-xs pr-10 ${!showSecret ? '[&]:text-security-disc' : ''}`}
                        style={!showSecret ? { WebkitTextSecurity: 'disc' } as React.CSSProperties : undefined}
                        aria-describedby={fieldErrors.token ? 'err-token' : 'hint-token'}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="absolute right-1.5 top-1.5 size-7"
                        onClick={() => setShowSecret((v) => !v)}
                        aria-label={showSecret ? 'Hide secret' : 'Show secret'}
                      >
                        {showSecret ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                      </Button>
                    </div>
                  )}
                  <div className="flex items-center gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="h-7 text-xs"
                      onClick={() => fileInputRef.current?.click()}
                    >
                      <Upload className="size-3 mr-1" />
                      Upload file
                    </Button>
                    <input
                      ref={fileInputRef}
                      type="file"
                      className="hidden"
                      accept=".yaml,.yml,.json,.txt,.pem,.key,*"
                      onChange={handleFileUpload}
                    />
                    {providerMeta.tokenField.hint && (
                      <p id="hint-token" className="text-xs text-muted-foreground">{providerMeta.tokenField.hint}</p>
                    )}
                  </div>
                </>
              ) : (
                <div className="relative">
                  <Input
                    id="cred-token"
                    type={showSecret ? 'text' : 'password'}
                    placeholder="Enter token or API key"
                    value={token}
                    onChange={(e) => {
                      setToken(e.target.value)
                      if (fieldErrors.token) setFieldErrors((prev) => ({ ...prev, token: undefined }))
                    }}
                    autoComplete="off"
                    className="pr-10"
                    aria-describedby={fieldErrors.token ? 'err-token' : undefined}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute right-1 top-1/2 -translate-y-1/2 size-7"
                    onClick={() => setShowSecret((v) => !v)}
                    aria-label={showSecret ? 'Hide token' : 'Show token'}
                  >
                    {showSecret ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                  </Button>
                </div>
              )}
              {fieldErrors.token && (
                <p id="err-token" className="text-xs text-destructive">{fieldErrors.token}</p>
              )}
            </div>
          )}

          {/* URL */}
          {fields.includes('url') && (
            <div className="space-y-1.5">
              <label htmlFor="cred-url" className="text-sm font-medium">
                URL {!isUrlOptional && <span className="text-destructive">*</span>}
              </label>
              <Input
                id="cred-url"
                type="url"
                placeholder={
                  provider === 'github' ? 'https://github.example.com'
                    : provider === 'gitlab' ? 'https://gitlab.example.com'
                    : 'https://your-instance.atlassian.net'
                }
                value={url}
                onChange={(e) => {
                  setUrl(e.target.value)
                  if (fieldErrors.url) setFieldErrors((prev) => ({ ...prev, url: undefined }))
                }}
                aria-describedby={fieldErrors.url ? 'err-url' : isUrlOptional ? 'hint-url' : undefined}
              />
              {isUrlOptional && (
                <p id="hint-url" className="text-xs text-muted-foreground">{providerMeta?.urlHint}</p>
              )}
              {fieldErrors.url && (
                <p id="err-url" className="text-xs text-destructive">{fieldErrors.url}</p>
              )}
            </div>
          )}

          {/* Email */}
          {fields.includes('email') && (
            <div className="space-y-1.5">
              <label htmlFor="cred-email" className="text-sm font-medium">
                Email <span className="text-destructive">*</span>
              </label>
              <Input
                id="cred-email"
                type="email"
                placeholder="user@example.com"
                value={email}
                onChange={(e) => {
                  setEmail(e.target.value)
                  if (fieldErrors.email) setFieldErrors((prev) => ({ ...prev, email: undefined }))
                }}
                aria-describedby={fieldErrors.email ? 'err-email' : undefined}
              />
              {fieldErrors.email && (
                <p id="err-email" className="text-xs text-destructive">{fieldErrors.email}</p>
              )}
            </div>
          )}

          {/* Description */}
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

          {submitError && <p className="text-sm text-destructive">{submitError}</p>}

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
