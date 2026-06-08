export type AnnotationCategory =
  | 'ui'
  | 'integration'
  | 'review'
  | 'provenance'
  | 'cost'
  | 'oncall'
  | 'agent'

export type RegisteredAnnotation = {
  key: string
  category: AnnotationCategory
  label: string
  icon?: string
}

const ANNOTATION_REGISTRY: readonly RegisteredAnnotation[] = [
  // ui
  { key: 'ambient-code.io/ui/path', category: 'ui', label: 'Path' },
  { key: 'ambient-code.io/ui/pinned', category: 'ui', label: 'Pinned', icon: 'pin' },
  { key: 'ambient-code.io/ui/priority', category: 'ui', label: 'Priority', icon: 'alert-triangle' },
  { key: 'ambient-code.io/ui/tag', category: 'ui', label: 'Tag', icon: 'tag' },
  { key: 'ambient-code.io/ui/preview-url', category: 'ui', label: 'Preview URL', icon: 'external-link' },
  { key: 'ambient-code.io/ui/preview-title', category: 'ui', label: 'Preview Title' },

  // integration
  { key: 'ambient-code.io/jira/issue', category: 'integration', label: 'Jira Issue', icon: 'ticket' },
  { key: 'ambient-code.io/jira/epic', category: 'integration', label: 'Jira Epic', icon: 'layers' },
  { key: 'ambient-code.io/github/pr', category: 'integration', label: 'GitHub PR', icon: 'git-pull-request' },
  { key: 'ambient-code.io/github/repo', category: 'integration', label: 'GitHub Repo', icon: 'folder-git-2' },
  { key: 'ambient-code.io/github/branch', category: 'integration', label: 'GitHub Branch', icon: 'git-branch' },
  { key: 'ambient-code.io/gitlab/mr', category: 'integration', label: 'GitLab MR', icon: 'git-pull-request' },
  { key: 'ambient-code.io/gerrit/change', category: 'integration', label: 'Gerrit Change', icon: 'git-pull-request' },

  // review
  { key: 'ambient-code.io/review/status', category: 'review', label: 'Review Status', icon: 'message-circle' },
  { key: 'ambient-code.io/review/reviewer', category: 'review', label: 'Reviewer', icon: 'user' },

  // provenance
  { key: 'ambient-code.io/triggered-by', category: 'provenance', label: 'Triggered By', icon: 'play' },

  // cost
  { key: 'ambient-code.io/cost/estimate', category: 'cost', label: 'Cost Estimate', icon: 'dollar-sign' },

  // oncall
  { key: 'ambient-code.io/oncall/incident', category: 'oncall', label: 'Incident', icon: 'siren' },

  // agent
  { key: 'ambient-code.io/parent-agent', category: 'agent', label: 'Parent Agent', icon: 'bot' },
] as const

const registryByKey = new Map<string, RegisteredAnnotation>(
  ANNOTATION_REGISTRY.map((entry) => [entry.key, entry])
)

export function getRegisteredAnnotation(key: string): RegisteredAnnotation | null {
  return registryByKey.get(key) ?? null
}

export function isRegisteredAnnotation(key: string): boolean {
  return registryByKey.has(key)
}

export function getAnnotationsByCategory(category: AnnotationCategory): RegisteredAnnotation[] {
  return ANNOTATION_REGISTRY.filter((entry) => entry.category === category)
}

export function getRegisteredAnnotations(
  annotations: Record<string, string>
): Array<{ annotation: RegisteredAnnotation; value: string }> {
  const results: Array<{ annotation: RegisteredAnnotation; value: string }> = []

  for (const [key, value] of Object.entries(annotations)) {
    const registration = registryByKey.get(key)
    if (registration) {
      results.push({ annotation: registration, value })
    }
  }

  return results
}

const PREVIEW_URL_KEY = 'ambient-code.io/ui/preview-url'
const PREVIEW_TITLE_KEY = 'ambient-code.io/ui/preview-title'

export function getPreviewAnnotations(
  annotations: Record<string, string>
): { url: string; title?: string } | null {
  const url = annotations[PREVIEW_URL_KEY]
  if (!url) {
    return null
  }

  const title = annotations[PREVIEW_TITLE_KEY]
  return title ? { url, title } : { url }
}
