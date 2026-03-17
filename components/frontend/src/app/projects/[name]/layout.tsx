"use client";

import { useEffect, useMemo } from "react";
import { useParams, useRouter, usePathname } from "next/navigation";
import { PanelLeft, Plug, LogOut } from "lucide-react";
import Link from "next/link";
import { useVersion } from "@/services/queries/use-version";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { ThemeToggle } from "@/components/theme-toggle";
import { UserBubble } from "@/components/user-bubble";
import { cn } from "@/lib/utils";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { SessionsSidebar } from "./sessions/[sessionName]/components/sessions-sidebar";

export default function ProjectLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const params = useParams();
  const router = useRouter();
  const pathname = usePathname();
  const projectName = params?.name as string;

  // Extract session name from URL: /projects/{name}/sessions/{sessionName}
  const currentSessionName = useMemo(() => {
    if (!pathname) return "";
    const match = pathname.match(/\/sessions\/([^/]+)/);
    return match ? decodeURIComponent(match[1]) : "";
  }, [pathname]);
  const [sidebarVisible, setSidebarVisible] = useLocalStorage(
    "session-sidebar-visible",
    true
  );
  const { data: version } = useVersion();

  const handleLogout = () => {
    window.location.href = '/oauth/sign_out';
  };

  // Persist last visited project for redirect on next visit
  useEffect(() => {
    if (projectName) {
      try { localStorage.setItem("selectedProject", projectName); } catch {}
    }
  }, [projectName]);

  if (!projectName) return null;

  return (
    <div className="absolute inset-0 overflow-hidden bg-background flex flex-col">
      <div className="flex-grow overflow-hidden bg-card flex">
        {/* Left sidebar */}
        <div
          className={cn(
            "h-full overflow-hidden border-r transition-[width] duration-200 ease-in-out flex-shrink-0",
            sidebarVisible ? "w-[300px]" : "w-0 border-r-0"
          )}
        >
          <div className="h-full w-[280px]">
            <SessionsSidebar
              projectName={projectName}
              currentSessionName={currentSessionName}
              collapsed={false}
              onCollapse={() => setSidebarVisible(false)}
            />
          </div>
        </div>

        {/* Main content */}
        <div className="flex-1 min-w-0 flex flex-col h-full">
          {/* Content header with nav items */}
          <div className="flex-shrink-0 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <div className="flex h-14 items-center justify-between gap-3 px-4">
              {/* Left: branding when sidebar is collapsed */}
              <div className="flex items-center gap-2">
                {!sidebarVisible && (
                  <>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setSidebarVisible(true)}
                      className="h-8 w-8 p-0"
                      title="Show sessions sidebar"
                    >
                      <PanelLeft className="h-4 w-4" />
                    </Button>
                    <Link href="/" className="flex items-end gap-2">
                      <span className="text-lg font-bold">Ambient Code Platform</span>
                      {version && (
                        <span className="text-[0.65rem] text-muted-foreground/60 pb-0.5">
                          {version}
                        </span>
                      )}
                    </Link>
                  </>
                )}
              </div>

              {/* Right: nav items */}
              <div className="flex items-center gap-3">
                <ThemeToggle />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => router.push('/integrations')}
                  className="text-muted-foreground hover:text-foreground"
                >
                  <Plug className="w-4 h-4 mr-1" />
                  Integrations
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger className="outline-none">
                    <UserBubble />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onSelect={handleLogout}>
                      <LogOut className="w-4 h-4 mr-2" />
                      Logout
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          </div>

          {/* Page content */}
          <div className="flex-1 overflow-auto">
            {children}
          </div>
        </div>
      </div>
    </div>
  );
}
