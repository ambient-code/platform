import { NextRequest, NextResponse } from "next/server";
import { BACKEND_URL } from "@/lib/config";
import { buildForwardHeadersAsync } from "@/lib/auth";

/**
 * GET /api/projects/:name/learned
 *
 * Reads docs/learned/ from the workspace repo via the backend repo tree/blob
 * endpoints. Parses frontmatter from each markdown file to build entries.
 */
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name: projectName } = await params;
    const headers = await buildForwardHeadersAsync(request);
    const searchParams = request.nextUrl.searchParams;
    const typeFilter = searchParams.get("type") || "";
    const page = parseInt(searchParams.get("page") || "0", 10);
    const pageSize = parseInt(searchParams.get("pageSize") || "50", 10);

    // Get the project to find repo annotation
    const projectRes = await fetch(
      `${BACKEND_URL}/projects/${projectName}`,
      { method: "GET", headers }
    );
    if (!projectRes.ok) {
      return NextResponse.json({ entries: [], totalCount: 0 });
    }
    const project = await projectRes.json();
    const repoAnnotation =
      project?.data?.annotations?.["ambient.ai/repo"] ||
      project?.annotations?.["ambient.ai/repo"] ||
      "";

    if (!repoAnnotation) {
      return NextResponse.json({ entries: [], totalCount: 0 });
    }

    // Fetch tree for docs/learned/corrections/ and docs/learned/patterns/
    const types = typeFilter ? [typeFilter] : ["correction", "pattern"];
    type ParsedEntry = {
      title: string;
      type: "correction" | "pattern";
      date: string;
      author: string;
      contentPreview: string;
      filePath: string;
      source: string;
      session: string;
    };
    const allEntries: ParsedEntry[] = [];

    for (const t of types) {
      const dir = `docs/learned/${t}s`;
      const treeParams = new URLSearchParams({
        repo: repoAnnotation,
        ref: "HEAD",
        path: dir,
      });

      const treeRes = await fetch(
        `${BACKEND_URL}/projects/${projectName}/repo/tree?${treeParams.toString()}`,
        { method: "GET", headers }
      );

      if (!treeRes.ok) continue;

      const treeData = await treeRes.json();
      const entries =
        treeData?.data?.entries || treeData?.entries || [];

      for (const entry of entries as Array<{ path?: string; name?: string }>) {
        const fileName = entry.path || entry.name || "";
        if (!fileName.endsWith(".md")) continue;

        const blobParams = new URLSearchParams({
          repo: repoAnnotation,
          ref: "HEAD",
          path: `${dir}/${fileName}`,
        });

        try {
          const blobRes = await fetch(
            `${BACKEND_URL}/projects/${projectName}/repo/blob?${blobParams.toString()}`,
            { method: "GET", headers }
          );

          if (!blobRes.ok) continue;

          const blobText = await blobRes.text();
          const parsed = parseFrontmatter(
            blobText,
            `${dir}/${fileName}`,
            t as "correction" | "pattern"
          );
          if (parsed) {
            allEntries.push(parsed);
          }
        } catch {
          continue;
        }
      }
    }

    // Sort by date descending
    allEntries.sort((a, b) => b.date.localeCompare(a.date));

    // Paginate
    const totalCount = allEntries.length;
    const start = page * pageSize;
    const paged = allEntries.slice(start, start + pageSize);

    return NextResponse.json({ entries: paged, totalCount });
  } catch (error) {
    console.error("Failed to fetch learned files:", error);
    return NextResponse.json({ entries: [], totalCount: 0 });
  }
}

/**
 * Parse YAML frontmatter from a markdown file.
 */
function parseFrontmatter(
  content: string,
  filePath: string,
  fallbackType: "correction" | "pattern"
): {
  title: string;
  type: "correction" | "pattern";
  date: string;
  author: string;
  contentPreview: string;
  filePath: string;
  source: string;
  session: string;
} | null {
  const fmMatch = content.match(/^---\s*\n([\s\S]*?)\n---\s*\n([\s\S]*)/);
  if (!fmMatch) {
    const filename = filePath.split("/").pop() || "";
    const bodyContent = content.trim();
    return {
      title: filename.replace(/\.md$/, ""),
      type: fallbackType,
      date: "",
      author: "",
      contentPreview: bodyContent.slice(0, 200),
      filePath,
      source: "",
      session: "",
    };
  }

  const frontmatterBlock = fmMatch[1];
  const body = fmMatch[2].trim();

  // Simple YAML key: value parsing (no nested structures needed)
  const fm: Record<string, string> = {};
  for (const line of frontmatterBlock.split("\n")) {
    const colonIdx = line.indexOf(":");
    if (colonIdx === -1) continue;
    const key = line.slice(0, colonIdx).trim();
    const value = line.slice(colonIdx + 1).trim();
    fm[key] = value;
  }

  const validTypes = ["correction", "pattern"];
  const parsedType = validTypes.includes(fm.type)
    ? (fm.type as "correction" | "pattern")
    : fallbackType;

  return {
    title: fm.title || filePath.split("/").pop()?.replace(/\.md$/, "") || "",
    type: parsedType,
    date: fm.date || "",
    author: fm.author || fm.source || "",
    contentPreview: body.slice(0, 200),
    filePath,
    source: fm.source || "",
    session: fm.session || "",
  };
}
