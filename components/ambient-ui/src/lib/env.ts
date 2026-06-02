function getEnv(key: string, defaultValue: string): string {
  return process.env[key] || defaultValue
}

function getOptionalEnv(key: string): string | undefined {
  return process.env[key] || undefined
}

export const env = {
  API_SERVER_URL: getEnv('API_SERVER_URL', 'http://localhost:8000'),
  SSO_ISSUER_URL: getOptionalEnv('SSO_ISSUER_URL'),
  SSO_CLIENT_ID: getOptionalEnv('SSO_CLIENT_ID'),
  SSO_CLIENT_SECRET: getOptionalEnv('SSO_CLIENT_SECRET'),
  SSO_REDIRECT_URI: getOptionalEnv('SSO_REDIRECT_URI'),
  SESSION_SECRET: getOptionalEnv('SESSION_SECRET'),
} as const
