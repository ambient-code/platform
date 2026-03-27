import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ instanceName: string }> }
) {
  const { instanceName } = await params
  const safeInstanceName = encodeURIComponent(instanceName)
  const headers = await buildForwardHeadersAsync(request)

  const resp = await fetch(`${BACKEND_URL}/auth/gerrit/${safeInstanceName}/disconnect`, {
    method: 'DELETE',
    headers,
  })

  const data = await resp.text()
  return new Response(data, { status: resp.status, headers: { 'Content-Type': 'application/json' } })
}
