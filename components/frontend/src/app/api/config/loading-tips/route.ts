import { env } from '@/lib/env';
import { DEFAULT_LOADING_TIPS } from '@/lib/loading-tips';

export async function GET() {
  let tips = DEFAULT_LOADING_TIPS;

  if (env.LOADING_TIPS) {
    try {
      const parsed = JSON.parse(env.LOADING_TIPS);
      if (Array.isArray(parsed) && parsed.length > 0 && parsed.every(t => typeof t === 'string')) {
        tips = parsed;
      }
    } catch {
      // Invalid JSON, fall back to defaults
      console.warn('LOADING_TIPS environment variable contains invalid JSON, using defaults');
    }
  }

  return Response.json({ tips });
}
