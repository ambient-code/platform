import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

/**
 * Creates GET and PUT route handlers that proxy requests to the backend.
 * @param backendPath - Function that takes the project name and returns the backend URL path.
 */
export function createProxyRouteHandlers(backendPath: (name: string) => string) {
  return {
    GET: async (request: Request, { params }: { params: Promise<{ name: string }> }) => {
      try {
        const { name } = await params;
        const headers = await buildForwardHeadersAsync(request);
        const response = await fetch(`${BACKEND_URL}${backendPath(name)}`, { headers });
        const text = await response.text();
        return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
      } catch (error) {
        console.error(`Error proxying GET ${backendPath('...')}:`, error);
        return Response.json({ error: 'Failed to fetch resource' }, { status: 500 });
      }
    },
    PUT: async (request: Request, { params }: { params: Promise<{ name: string }> }) => {
      try {
        const { name } = await params;
        const body = await request.text();
        const headers = await buildForwardHeadersAsync(request);
        const response = await fetch(`${BACKEND_URL}${backendPath(name)}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', ...headers },
          body,
        });
        const text = await response.text();
        return new Response(text, { status: response.status, headers: { 'Content-Type': 'application/json' } });
      } catch (error) {
        console.error(`Error proxying PUT ${backendPath('...')}:`, error);
        return Response.json({ error: 'Failed to update resource' }, { status: 500 });
      }
    },
  };
}
