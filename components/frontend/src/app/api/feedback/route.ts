import { NextRequest, NextResponse } from 'next/server';
import { getLangfuseClient } from '@/lib/langfuseClient';

/**
 * POST /api/feedback
 * 
 * Sends user feedback to Langfuse as a score.
 * This route builds rich context from the session and sends it to Langfuse
 * using the LangfuseWeb SDK (public key only - no secret key needed).
 * 
 * Request body:
 * - traceId: string (optional - if we have a trace ID from the session)
 * - value: number (1 for positive, 0 for negative)
 * - comment?: string (optional user comment)
 * - username: string
 * - projectName: string
 * - sessionName: string
 * - workflow?: string (optional - active workflow name)
 * - context?: string (what the user was working on)
 * - includeTranscript?: boolean
 * - transcript?: Array<{ role: string; content: string; timestamp?: string }>
 */

type FeedbackRequest = {
  traceId?: string;
  value: number;
  comment?: string;
  username: string;
  projectName: string;
  sessionName: string;
  workflow?: string;
  context?: string;
  includeTranscript?: boolean;
  transcript?: Array<{ role: string; content: string; timestamp?: string }>;
};

/**
 * Sanitize a string to prevent log injection attacks.
 * Removes control characters that could be used to fake log entries.
 */
function sanitizeString(input: string): string {
  // Remove control characters except newlines and tabs (which we'll normalize)
  // This prevents log injection via carriage returns, null bytes, etc.
  return input
    .replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, '') // Remove control chars except \t \n \r
    .replace(/\r\n/g, '\n') // Normalize line endings
    .replace(/\r/g, '\n');
}

export async function POST(request: NextRequest) {
  try {
    const body: FeedbackRequest = await request.json();
    
    const {
      traceId,
      value,
      comment,
      username,
      projectName,
      sessionName,
      workflow,
      context,
      includeTranscript,
      transcript,
    } = body;

    // Validate required fields
    if (typeof value !== 'number' || !username || !projectName || !sessionName) {
      return NextResponse.json(
        { error: 'Missing required fields: value, username, projectName, sessionName' },
        { status: 400 }
      );
    }

    // Validate value range (must be 0 or 1 for thumbs down/up)
    if (value !== 0 && value !== 1) {
      return NextResponse.json(
        { error: 'Invalid value: must be 0 (negative) or 1 (positive)' },
        { status: 400 }
      );
    }

    // Sanitize string inputs to prevent log injection
    const sanitizedUsername = sanitizeString(username);
    const sanitizedProjectName = sanitizeString(projectName);
    const sanitizedSessionName = sanitizeString(sessionName);
    const sanitizedComment = comment ? sanitizeString(comment) : undefined;
    const sanitizedWorkflow = workflow ? sanitizeString(workflow) : undefined;
    const sanitizedContext = context ? sanitizeString(context) : undefined;
    const sanitizedTraceId = traceId ? sanitizeString(traceId) : undefined;

    // Get Langfuse client (uses public key only)
    const langfuse = getLangfuseClient();

    if (!langfuse) {
      console.warn('Langfuse not configured - feedback will not be recorded');
      return NextResponse.json({ 
        success: false, 
        message: 'Langfuse not configured' 
      });
    }

    // Sanitize transcript entries if provided
    const sanitizedTranscript = transcript?.map(m => ({
      role: sanitizeString(m.role),
      content: sanitizeString(m.content),
      timestamp: m.timestamp ? sanitizeString(m.timestamp) : undefined,
    }));

    // Build the feedback comment (user-provided content only)
    const commentParts: string[] = [];
    
    if (sanitizedComment) {
      commentParts.push(sanitizedComment);
    }
    
    if (sanitizedContext) {
      commentParts.push(`\nMessage:\n${sanitizedContext}`);
    }
    
    if (includeTranscript && sanitizedTranscript && sanitizedTranscript.length > 0) {
      const transcriptText = sanitizedTranscript
        .map(m => `[${m.role}]: ${m.content}`)
        .join('\n');
      commentParts.push(`\nFull Transcript:\n${transcriptText}`);
    }

    const feedbackComment = commentParts.length > 0 ? commentParts.join('\n') : undefined;

    // Determine the traceId to use
    const effectiveTraceId = sanitizedTraceId || `feedback-${sanitizedSessionName}-${Date.now()}`;

    // Build metadata with structured session info
    const metadata: Record<string, string> = {
      project: sanitizedProjectName,
      session: sanitizedSessionName,
      user: sanitizedUsername,
    };
    
    if (sanitizedWorkflow) {
      metadata.workflow = sanitizedWorkflow;
    }

    // Send feedback using LangfuseWeb SDK
    langfuse.score({
      traceId: effectiveTraceId,
      name: 'user-feedback',
      value: value,
      comment: feedbackComment,
      metadata,
    });

    return NextResponse.json({ 
      success: true, 
      traceId: effectiveTraceId,
      message: 'Feedback submitted successfully' 
    });

  } catch (error) {
    console.error('Error submitting feedback:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
