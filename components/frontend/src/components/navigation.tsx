"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { UserBubble } from "@/components/user-bubble";
import { ThemeToggle } from "@/components/theme-toggle";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Plug, LogOut } from "lucide-react";
import { useVersion } from "@/services/queries/use-version";

type NavigationProps = {
  feedbackUrl?: string;
};

export function Navigation({ feedbackUrl }: NavigationProps) {
  // const pathname = usePathname();
  // const segments = pathname?.split("/").filter(Boolean) || [];
  const router = useRouter();
  const { data: version } = useVersion();

  const handleLogout = () => {
    // Redirect to oauth-proxy logout endpoint  
    // This clears the OpenShift OAuth session and redirects back to login  
    window.location.href = '/oauth/sign_out';  
  };

  return (
    <nav className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="px-4">
        <div className="flex h-10 items-center justify-between gap-3">
          <div className="flex items-end gap-1.5">
            <Link href="/" className="text-base font-bold">
              <span className="hidden md:inline">Ambient Code Platform</span>
              <span className="md:hidden">ACP</span>
            </Link>
            {version && (
              <a
                href="https://github.com/ambient-code/platform/releases"
                target="_blank"
                rel="noopener noreferrer"
                className="text-[0.6rem] text-muted-foreground/60 pb-0.5 hover:text-muted-foreground transition-colors"
              >
                <span>{version}</span>
              </a>
            )}
          </div>
          <div className="flex items-center gap-2">
            {feedbackUrl && (
              <a
                href={feedbackUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                Feedback
              </a>
            )}
            <ThemeToggle />
            <DropdownMenu>
              <DropdownMenuTrigger className="outline-none">
                <UserBubble />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onSelect={() => router.push('/integrations')}>
                  <Plug className="w-4 h-4 mr-2" />
                  Integrations
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={handleLogout}>
                  <LogOut className="w-4 h-4 mr-2" />
                  Logout
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </nav>
  );
}