'use client'

import React, { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Loader2, Eye, EyeOff, Plug } from 'lucide-react'
import { toast } from 'sonner'
import {
  useConnectMCPServer,
  useDisconnectMCPServer,
} from '@/services/queries/use-mcp-credentials'

type FieldDefinition = {
  name: string
  label: string
  type: 'text' | 'password'
  placeholder?: string
  helpText?: string
}

type MCPServerStatusData = {
  connected: boolean
  serverName: string
  fieldNames?: string[]
  updatedAt?: string
}

type Props = {
  serverName: string
  displayName: string
  description: string
  fields: FieldDefinition[]
  status?: MCPServerStatusData
  onRefresh?: () => void
}

export function MCPCredentialCard({
  serverName,
  displayName,
  description,
  fields,
  status,
  onRefresh,
}: Props) {
  const connectMutation = useConnectMCPServer(serverName)
  const disconnectMutation = useDisconnectMCPServer(serverName)
  const isLoading = !status

  const [showForm, setShowForm] = useState(false)
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({})
  const [visibleFields, setVisibleFields] = useState<Record<string, boolean>>({})

  const handleConnect = () => {
    const missingFields = fields.filter((f) => !fieldValues[f.name]?.trim())
    if (missingFields.length > 0) {
      toast.error('Please fill in all fields')
      return
    }

    connectMutation.mutate(
      { fields: fieldValues },
      {
        onSuccess: () => {
          toast.success(`${displayName} connected successfully`)
          setShowForm(false)
          setFieldValues({})
          onRefresh?.()
        },
        onError: (error) => {
          toast.error(
            error instanceof Error ? error.message : `Failed to connect ${displayName}`
          )
        },
      }
    )
  }

  const handleDisconnect = () => {
    disconnectMutation.mutate(undefined, {
      onSuccess: () => {
        toast.success(`${displayName} disconnected successfully`)
        onRefresh?.()
      },
      onError: (error) => {
        toast.error(
          error instanceof Error ? error.message : `Failed to disconnect ${displayName}`
        )
      },
    })
  }

  const toggleFieldVisibility = (fieldName: string) => {
    setVisibleFields((prev) => ({ ...prev, [fieldName]: !prev[fieldName] }))
  }

  return (
    <Card className="bg-card border border-border/60 shadow-sm shadow-black/[0.03] dark:shadow-black/[0.15] flex flex-col h-full">
      <div className="p-6 flex flex-col flex-1">
        {/* Header */}
        <div className="flex items-start gap-4 mb-6">
          <div className="flex-shrink-0 w-16 h-16 bg-primary rounded-lg flex items-center justify-center">
            <Plug className="w-8 h-8 text-white" />
          </div>
          <div className="flex-1">
            <h3 className="text-xl font-semibold text-foreground mb-1">{displayName}</h3>
            <p className="text-muted-foreground">{description}</p>
          </div>
        </div>

        {/* Status */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span
              className={`w-2 h-2 rounded-full ${status?.connected ? 'bg-green-500' : 'bg-gray-400'}`}
            />
            <span className="text-sm font-medium text-foreground/80">
              {status?.connected ? 'Connected' : 'Not Connected'}
            </span>
          </div>
        </div>

        {/* Form */}
        {showForm && (
          <div className="mb-4 space-y-3">
            {fields.map((field) => (
              <div key={field.name}>
                <Label htmlFor={`mcp-${serverName}-${field.name}`} className="text-sm">
                  {field.label}
                </Label>
                <div className="flex gap-2 mt-1">
                  <Input
                    id={`mcp-${serverName}-${field.name}`}
                    type={
                      field.type === 'password' && !visibleFields[field.name]
                        ? 'password'
                        : 'text'
                    }
                    placeholder={field.placeholder}
                    value={fieldValues[field.name] ?? ''}
                    onChange={(e) =>
                      setFieldValues((prev) => ({
                        ...prev,
                        [field.name]: e.target.value,
                      }))
                    }
                    disabled={connectMutation.isPending}
                  />
                  {field.type === 'password' && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => toggleFieldVisibility(field.name)}
                      disabled={connectMutation.isPending}
                    >
                      {visibleFields[field.name] ? (
                        <EyeOff className="w-4 h-4" />
                      ) : (
                        <Eye className="w-4 h-4" />
                      )}
                    </Button>
                  )}
                </div>
                {field.helpText && (
                  <p className="text-xs text-muted-foreground mt-1">{field.helpText}</p>
                )}
              </div>
            ))}
            <div className="flex gap-2 pt-2">
              <Button
                onClick={handleConnect}
                disabled={
                  connectMutation.isPending ||
                  fields.some((f) => !fieldValues[f.name]?.trim())
                }
                className="flex-1"
              >
                {connectMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Connecting...
                  </>
                ) : (
                  'Save Credentials'
                )}
              </Button>
              <Button
                variant="outline"
                onClick={() => setShowForm(false)}
                disabled={connectMutation.isPending}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        {/* Action buttons */}
        <div className="flex gap-3 mt-auto">
          {status?.connected && !showForm ? (
            <>
              <Button
                variant="outline"
                onClick={() => setShowForm(true)}
                disabled={isLoading || disconnectMutation.isPending}
              >
                Edit
              </Button>
              <Button
                variant="destructive"
                onClick={handleDisconnect}
                disabled={isLoading || disconnectMutation.isPending}
              >
                {disconnectMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Disconnecting...
                  </>
                ) : (
                  'Disconnect'
                )}
              </Button>
            </>
          ) : !showForm ? (
            <Button
              onClick={() => setShowForm(true)}
              disabled={isLoading}
              className="bg-primary hover:bg-primary/90 text-primary-foreground"
            >
              Connect {displayName}
            </Button>
          ) : null}
        </div>
      </div>
    </Card>
  )
}
