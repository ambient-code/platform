import { NextRequest, NextResponse } from "next/server";
import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

/**
 * POST /api/projects/:name/learned/create
 *
 * Creates a new memory by opening a PR in the workspace repo.
 */
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const body = await request.json();

    const { title, content, type, repo } = body as {
      title: string;
      content: string;
      type: "correction" | "pattern";
      repo?: string;
    };

    if (!title || !content || !type) {
      return NextResponse.json(
        { error: "title, content, and type are required" },
        { status: 400 }
      );
    }

    if (!["correction", "pattern"].includes(type)) {
      return NextResponse.json(
        { error: "type must be 'correction' or 'pattern'" },
        { status: 400 }
      );
    }

    // Resolve repo: use explicit repo from request, fall back to project annotation
    let repoAnnotation = repo || "";
    if (!repoAnnotation) {
      const projectRes = await fetch(
        `${BACKEND_URL}/projects/${projectName}`,
        { method: "GET", headers }
      );
      if (projectRes.ok) {
        const project = await projectRes.json();
        repoAnnotation =
          project?.data?.annotations?.["ambient.ai/repo"] ||
          project?.annotations?.["ambient.ai/repo"] ||
          "";
      }
    }

    if (!repoAnnotation) {
      return NextResponse.json(
        { error: "No repository specified. Enter a target repository (owner/repo)." },
        { status: 400 }
      );
    }

    const ownerRepo = parseOwnerRepo(repoAnnotation);
    if (!ownerRepo) {
      return NextResponse.json(
        { error: "Invalid repository format. Use owner/repo (e.g. jeremyeder/continuous-learning-example)" },
        { status: 400 }
      );
    }

    const date = new Date().toISOString().split("T")[0];
    const slug = title
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .slice(0, 60);
    const branchName = `learned/${type}-${date}-${slug}`;

    const prData = {
      owner: ownerRepo.owner,
      repo: ownerRepo.repo,
      title: `learned: ${title}`,
      body: `## New Memory\n\n**Type:** ${type}\n**Source:** Manual entry\n\n---\n\n${content}`,
      head: branchName,
      base: "main",
      draft: false,
    };

    const prRes = await fetch(
      `${BACKEND_URL}/projects/${projectName}/github/pr`,
      {
        method: "POST",
        headers: { ...headers, "Content-Type": "application/json" },
        body: JSON.stringify(prData),
      }
    );

    if (!prRes.ok) {
      const errText = await prRes.text();
      return NextResponse.json(
        { error: `Failed to create PR: ${errText}` },
        { status: 500 }
      );
    }

    const prResult = await prRes.json();
    const resultData = prResult?.data || prResult;

    return NextResponse.json({
      prUrl: resultData.url || "",
      prNumber: resultData.number || 0,
    });
  } catch (error) {
    console.error("Failed to create memory:", error);
    return NextResponse.json(
      { error: "Failed to create memory" },
      { status: 500 }
    );
  }
}

import { parseOwnerRepo } from "@/lib/github-utils";
