import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Workspaces · Ambient Code Platform",
};

export default function WorkspacesLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
