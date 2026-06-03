'use client'

import { useCallback, useMemo, useState } from 'react'
import { Copy, Download, Check } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import type { DomainAgent } from '@/domain/types'

function agentToYaml(agent: DomainAgent): string {
  const lines: string[] = [
    'apiVersion: ambient-code.io/v1',
    'kind: Agent',
    'metadata:',
    `  name: ${agent.name}`,
  ]

  const annotationEntries = Object.entries(agent.annotations)
  if (annotationEntries.length > 0) {
    lines.push('  annotations:')
    for (const [key, value] of annotationEntries) {
      lines.push(`    ${key}: "${value}"`)
    }
  }

  const labelEntries = Object.entries(agent.labels)
  if (labelEntries.length > 0) {
    lines.push('  labels:')
    for (const [key, value] of labelEntries) {
      lines.push(`    ${key}: "${value}"`)
    }
  }

  lines.push('spec:')

  if (agent.displayName) {
    lines.push(`  displayName: "${agent.displayName}"`)
  }
  if (agent.description) {
    lines.push(`  description: "${agent.description}"`)
  }
  if (agent.model) {
    lines.push(`  model: ${agent.model}`)
  }
  if (agent.repoUrl) {
    lines.push(`  repoUrl: ${agent.repoUrl}`)
  }
  if (agent.workflowId) {
    lines.push(`  workflowId: ${agent.workflowId}`)
  }
  if (agent.prompt) {
    lines.push('  prompt: |')
    for (const promptLine of agent.prompt.split('\n')) {
      lines.push(`    ${promptLine}`)
    }
  }

  return lines.join('\n') + '\n'
}

export function AgentConfigTab({ agent }: { agent: DomainAgent }) {
  const [copied, setCopied] = useState(false)
  const yaml = useMemo(() => agentToYaml(agent), [agent])

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(yaml)
    setCopied(true)
    globalThis.setTimeout(() => setCopied(false), 2000)
  }, [yaml])

  const handleDownload = useCallback(() => {
    const blob = new Blob([yaml], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `agent-${agent.name}.yaml`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }, [yaml, agent.name])

  return (
    <div className="space-y-6 pt-4">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">Agent Definition (YAML)</CardTitle>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopy}
              >
                {copied ? (
                  <>
                    <Check className="size-4 mr-1.5" />
                    Copied
                  </>
                ) : (
                  <>
                    <Copy className="size-4 mr-1.5" />
                    Copy to Clipboard
                  </>
                )}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleDownload}
              >
                <Download className="size-4 mr-1.5" />
                Download YAML
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <pre className="whitespace-pre-wrap rounded-md bg-muted p-4 text-sm font-mono overflow-x-auto">
            {yaml}
          </pre>
        </CardContent>
      </Card>
    </div>
  )
}
