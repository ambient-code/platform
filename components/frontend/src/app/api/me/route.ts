import { buildForwardHeadersAsync } from '@/lib/auth';

export async function GET(request: Request) {
  try {
    // Use the shared helper so dev oc whoami and env fallbacks apply uniformly
    const headers = await buildForwardHeadersAsync(request);
    const userId = headers['X-Forwarded-User'] || '';
    const email = headers['X-Forwarded-Email'] || '';
    const username = headers['X-Forwarded-Preferred-Username'] || '';
    const token = headers['X-Forwarded-Access-Token'] || '';

    if (!userId && !username && !email && !token) {
      return Response.json({ authenticated: false }, { status: 200 });
    }

    // Clean the displayName by removing cluster suffix (e.g., "@cluster.local", "@apps-crc.testing")
    const rawDisplayName = username || email || userId;
    const displayName = rawDisplayName?.split('@')[0] || rawDisplayName;

    return Response.json({
      authenticated: true,
      userId,
      email,
      username,
      displayName,
    });
  } catch (error) {
    console.error('Error reading user headers:', error);
    return Response.json({ authenticated: false }, { status: 200 });
  }
}


