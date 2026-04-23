import React from 'react'
import IntegrationsClient from '@/app/integrations/IntegrationsClient'

export const dynamic = 'force-dynamic'
export const revalidate = 0

export default function IntegrationsPage() {
  const appSlug = process.env.GITHUB_APP_SLUG
  const githubCallbackUrl = process.env.GITHUB_CALLBACK_URL
  return <IntegrationsClient appSlug={appSlug} githubCallbackUrl={githubCallbackUrl} />
}
