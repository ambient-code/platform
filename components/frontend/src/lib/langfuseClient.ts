import { LangfuseWeb } from "langfuse";

/**
 * Langfuse Web client for sending scores/feedback.
 * Uses only the public key (no secret key needed).
 * 
 * This client is used server-side in API routes to send feedback
 * while keeping the implementation simple and secure.
 */

let langfuseClient: LangfuseWeb | null = null;

export function getLangfuseClient(): LangfuseWeb | null {
  // Return cached client if available
  if (langfuseClient) {
    return langfuseClient;
  }

  const publicKey = process.env.LANGFUSE_PUBLIC_KEY;
  const host = process.env.LANGFUSE_HOST;

  if (!publicKey || !host) {
    console.warn('Langfuse not configured (missing LANGFUSE_PUBLIC_KEY or LANGFUSE_HOST)');
    return null;
  }

  langfuseClient = new LangfuseWeb({
    publicKey,
    baseUrl: host,
  });

  return langfuseClient;
}
