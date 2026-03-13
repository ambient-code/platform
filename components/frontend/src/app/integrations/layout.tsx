import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Integrations · Ambient Code Platform",
};

export default function IntegrationsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
