import { BACKEND_URL } from '@/lib/config';
import { buildForwardHeadersAsync } from '@/lib/auth';

/**
 * Categorizes an error and returns a structured error response.
 */
function getErrorResponse(error: unknown, operation: 'fetch' | 'update'): { message: string; status: number } {
  if (error instanceof TypeError && (error.message.includes('fetch') || error.message.includes('network'))) {
    return { message: `Backend service unavailable`, status: 503 };
  }
  if (error instanceof Error && error.name === 'AbortError') {
    return { message: 'Request timed out', status: 504 };
  }
  return { message: `Failed to ${operation} resource`, status: 500 };
}

/**
 * Logs proxy errors with structured context for debugging.
 */
function logProxyError(method: string, path: string, error: unknown): void {
  const errorType = error instanceof Error ? error.constructor.name : typeof error;
  const errorMessage = error instanceof Error ? error.message : String(error);
  console.error(`[proxy] ${method} ${path} failed: [${errorType}] ${errorMessage}`);
}

/**
 * Creates GET and PUT route handlers that proxy requests to the backend.
 * @param backendPath - Function that takes the project name and returns the backend URL path.
 * @returns Object with GET and PUT handlers following Next.js 15 dynamic route conventions.
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
        const path = backendPath('...');
        logProxyError('GET', path, error);
        const { message, status } = getErrorResponse(error, 'fetch');
        return Response.json({ error: message }, { status });
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
        const path = backendPath('...');
        logProxyError('PUT', path, error);
        const { message, status } = getErrorResponse(error, 'update');
        return Response.json({ error: message }, { status });
      }
    },
  };
}
