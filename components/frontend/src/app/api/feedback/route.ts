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

    // Get Langfuse client (uses public key only)
    const langfuse = getLangfuseClient();

    if (!langfuse) {
      console.warn('Langfuse not configured - feedback will not be recorded');
      return NextResponse.json({ 
        success: false, 
        message: 'Langfuse not configured' 
      });
    }

    // Build the feedback comment (user-provided content only)
    const commentParts: string[] = [];
    
    if (comment) {
      commentParts.push(comment);
    }
    
    if (context) {
      commentParts.push(`\nMessage:\n${context}`);
    }
    
    if (includeTranscript && transcript && transcript.length > 0) {
      const transcriptText = transcript
        .map(m => `[${m.role}]: ${m.content}`)
        .join('\n');
      commentParts.push(`\nFull Transcript:\n${transcriptText}`);
    }

    const feedbackComment = commentParts.length > 0 ? commentParts.join('\n') : undefined;

    // Determine the traceId to use
    const effectiveTraceId = traceId || `feedback-${sessionName}-${Date.now()}`;

    // Build metadata with structured session info
    const metadata: Record<string, string> = {
      project: projectName,
      session: sessionName,
      user: username,
    };
    
    if (workflow) {
      metadata.workflow = workflow;
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
