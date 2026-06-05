type CredentialField = 'token' | 'url' | 'email'

export type ProviderMeta = {
  provider: string
  label: string
  icon: string
  fields: CredentialField[]
}

export type CredentialCategory = {
  label: string
  providers: ProviderMeta[]
}

export const CREDENTIAL_CATEGORIES: readonly CredentialCategory[] = [
  {
    label: 'LLM Providers',
    providers: [
      { provider: 'anthropic', label: 'Anthropic', icon: 'Bot', fields: ['token'] },
      { provider: 'google-vertex', label: 'Google / Vertex', icon: 'Cloud', fields: ['token', 'url'] },
      { provider: 'openai', label: 'OpenAI', icon: 'Bot', fields: ['token'] },
    ],
  },
  {
    label: 'Source Control',
    providers: [
      { provider: 'github', label: 'GitHub', icon: 'Github', fields: ['token', 'url'] },
      { provider: 'gitlab', label: 'GitLab', icon: 'GitBranch', fields: ['token', 'url'] },
      { provider: 'gerrit', label: 'Gerrit', icon: 'GitBranch', fields: ['token', 'url'] },
    ],
  },
  {
    label: 'Project Management',
    providers: [
      { provider: 'jira', label: 'Jira', icon: 'Ticket', fields: ['token', 'email', 'url'] },
    ],
  },
  {
    label: 'Code Review',
    providers: [
      { provider: 'coderabbit', label: 'CodeRabbit', icon: 'Bot', fields: ['token'] },
    ],
  },
  {
    label: 'AI & Tooling',
    providers: [
      { provider: 'custom', label: 'Custom', icon: 'Key', fields: ['token', 'url', 'email'] },
    ],
  },
] as const

const providerIndex = new Map<string, ProviderMeta>()
const categoryIndex = new Map<string, string>()

for (const category of CREDENTIAL_CATEGORIES) {
  for (const provider of category.providers) {
    providerIndex.set(provider.provider, provider)
    categoryIndex.set(provider.provider, category.label)
  }
}

export function getProviderMeta(provider: string): ProviderMeta | undefined {
  return providerIndex.get(provider)
}

export function getCategoryForProvider(provider: string): string | undefined {
  return categoryIndex.get(provider)
}
