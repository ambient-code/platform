"use client";

import { usePathname } from "next/navigation";
import { Navigation } from "./navigation";

export function NavigationWrapper({ feedbackUrl }: { feedbackUrl?: string }) {
  const pathname = usePathname();
  // Hide on project sub-pages - they have their own split header
  if (pathname?.startsWith("/projects/") && pathname.split("/").length > 3) {
    return null;
  }
  return <Navigation feedbackUrl={feedbackUrl} />;
}
