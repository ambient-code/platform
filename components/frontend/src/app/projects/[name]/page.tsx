import { redirect } from "next/navigation";

export default async function ProjectDetailsPage({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = await params;
  redirect(`/projects/${encodeURIComponent(name)}/new`);
}
