import { BACKEND_URL } from '@/lib/config'
import { buildForwardHeadersAsync } from '@/lib/auth'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'

export async function POST(
  request: Request,
  { params }: { params: Promise<{ name: string; sessionName: string; taskId: string }> },
) {
  const { name, sessionName, taskId } = await params
  const headers = await buildForwardHeadersAsync(request)
  const body = await request.text()

  const resp = await fetch(
    `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/agui/tasks/${encodeURIComponent(taskId)}/stop`,
    {
      method: 'POST',
      headers: {
        ...headers,
        'Content-Type': 'application/json',
      },
      body,
    },
  )

  const data = await resp.text()
  return new Response(data, {
    status: resp.status,
    headers: { 'Content-Type': 'application/json' },
  })
}
