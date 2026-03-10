import type { Metadata } from 'next'
import React from 'react'
import IntegrationsClient from '@/app/integrations/IntegrationsClient'

export const metadata: Metadata = {
  title: 'Integrations · Ambient Code Platform',
}

export const dynamic = 'force-dynamic'
export const revalidate = 0

export default function IntegrationsPage() {
  const appSlug = process.env.GITHUB_APP_SLUG
  return <IntegrationsClient appSlug={appSlug} />
}
