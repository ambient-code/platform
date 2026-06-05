type CredentialField = 'token' | 'url' | 'email'

export type TokenFieldMeta = {
  label: string
  placeholder: string
  hint?: string
  multiline?: boolean
}

export type ProviderMeta = {
  provider: string
  label: string
  icon: string
  fields: CredentialField[]
  tokenField?: TokenFieldMeta
  namePlaceholder?: string
  urlOptional?: boolean
  urlHint?: string
}

export type CredentialCategory = {
  label: string
  providers: ProviderMeta[]
}

export const CREDENTIAL_CATEGORIES: readonly CredentialCategory[] = [
  {
    label: 'Source Control',
    providers: [
      { provider: 'github', label: 'GitHub', icon: 'Github', fields: ['token'],
        namePlaceholder: 'my-github-pat',
      },
      { provider: 'gitlab', label: 'GitLab', icon: 'GitBranch', fields: ['token'],
        namePlaceholder: 'my-gitlab-pat',
      },
    ],
  },
  {
    label: 'Project Management',
    providers: [
      { provider: 'jira', label: 'Jira', icon: 'Ticket', fields: ['token', 'email', 'url'],
        namePlaceholder: 'my-jira-token',
      },
    ],
  },
  {
    label: 'Cloud & Infrastructure',
    providers: [
      { provider: 'google', label: 'Google Cloud', icon: 'Cloud', fields: ['token'],
        namePlaceholder: 'my-gcp-service-account',
        tokenField: {
          label: 'Service Account Key (JSON)',
          placeholder: '{"type": "service_account", ...}',
          hint: 'Paste the full JSON key file for a GCP service account.',
          multiline: true,
        },
      },
      { provider: 'vertex', label: 'Vertex AI', icon: 'Cloud', fields: ['token'],
        namePlaceholder: 'my-vertex-service-account',
        tokenField: {
          label: 'Service Account Key (JSON)',
          placeholder: '{"type": "service_account", ...}',
          hint: 'Paste the full JSON key file for a GCP service account with Vertex AI API enabled.',
          multiline: true,
        },
      },
      { provider: 'kubeconfig', label: 'Kubernetes', icon: 'Server', fields: ['token'],
        namePlaceholder: 'my-cluster-kubeconfig',
        tokenField: {
          label: 'Kubeconfig',
          placeholder: 'apiVersion: v1\nkind: Config\nclusters:\n- ...',
          hint: 'Paste the entire contents of your kubeconfig file (~/.kube/config).',
          multiline: true,
        },
      },
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
