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
    label: 'Source Control',
    providers: [
      { provider: 'github', label: 'GitHub', icon: 'Github', fields: ['token', 'url'] },
      { provider: 'gitlab', label: 'GitLab', icon: 'GitBranch', fields: ['token', 'url'] },
    ],
  },
  {
    label: 'Project Management',
    providers: [
      { provider: 'jira', label: 'Jira', icon: 'Ticket', fields: ['token', 'email', 'url'] },
    ],
  },
  {
    label: 'Cloud & Infrastructure',
    providers: [
      { provider: 'google', label: 'Google Cloud', icon: 'Cloud', fields: ['token'] },
      { provider: 'vertex', label: 'Vertex AI', icon: 'Cloud', fields: ['token'] },
      { provider: 'kubeconfig', label: 'Kubernetes', icon: 'Server', fields: ['token'] },
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
