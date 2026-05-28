"use client";

import { useEffect, useMemo } from "react";
import { useParams, useRouter, usePathname } from "next/navigation";
import { PanelLeft, Plug, LogOut, Menu } from "lucide-react";
import Link from "next/link";
import { useVersion } from "@/services/queries/use-version";
import { useCurrentUser } from "@/services/queries";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";
import { ThemeToggle } from "@/components/theme-toggle";
import { RecentUpdatesDialog } from "@/components/recent-updates-dialog";
import { UserBubble } from "@/components/user-bubble";
import { cn } from "@/lib/utils";
import { useIsMobile } from "@/hooks/use-mobile";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { useResizePanel } from "@/hooks/use-resize-panel";
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
  const isMobile = useIsMobile();

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
  const [mobileSidebarOpen, setMobileSidebarOpen] = useLocalStorage(
    "session-sidebar-mobile-open",
    false
  );
  const sidebarResize = useResizePanel("session-sidebar-width", 280, 220, 450, "left");
  const { data: version } = useVersion();
  const { data: me } = useCurrentUser();

  const handleLogout = () => {
    if (me?.ssoEnabled) {
      window.location.href = '/api/auth/sso/logout';
    } else {
      window.location.href = '/oauth/sign_out';
    }
  };

  // Persist last visited project for redirect on next visit
  useEffect(() => {
    if (projectName) {
      try { localStorage.setItem("selectedProject", projectName); } catch {}
    }
  }, [projectName]);

  // Close mobile sidebar on navigation
  useEffect(() => {
    setMobileSidebarOpen(false);
  }, [pathname, setMobileSidebarOpen]);

  if (!projectName) return null;

  return (
    <div className="absolute inset-0 overflow-hidden bg-background flex flex-col">
      <div className="flex-grow overflow-hidden bg-card flex">
        {/* Mobile sidebar — Sheet overlay */}
        {isMobile && (
          <Sheet open={mobileSidebarOpen} onOpenChange={setMobileSidebarOpen}>
            <SheetContent side="left" className="w-[85vw] max-w-[320px] p-0" showCloseButton={false}>
              <SheetTitle className="sr-only">Sessions sidebar</SheetTitle>
              <SessionsSidebar
                projectName={projectName}
                currentSessionName={currentSessionName}
                collapsed={false}
                onCollapse={() => setMobileSidebarOpen(false)}
                onSessionSelect={() => setMobileSidebarOpen(false)}
              />
            </SheetContent>
          </Sheet>
        )}

        {/* Desktop sidebar */}
        {!isMobile && (
          <div
            className={cn(
              "h-full overflow-hidden border-r flex-shrink-0 relative",
              !sidebarResize.isDragging && "transition-[width] duration-200 ease-in-out",
              !sidebarVisible && "!w-0 border-r-0"
            )}
            style={{ width: sidebarVisible ? sidebarResize.width : 0 }}
          >
            <div className="h-full" style={{ width: sidebarResize.width }}>
              <SessionsSidebar
                projectName={projectName}
                currentSessionName={currentSessionName}
                collapsed={!sidebarVisible}
                onCollapse={() => setSidebarVisible(false)}
              />
            </div>
            {/* Drag handle */}
            <div
              className="absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-primary/50 transition-colors z-10"
              onMouseDown={sidebarResize.onMouseDown}
            />
          </div>
        )}

        {/* Main content */}
        <div className="flex-1 min-w-0 flex flex-col h-full">
          {/* Content header with nav items */}
          <div className="flex-shrink-0 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <div className="flex h-14 items-center justify-between gap-2 px-3 md:px-4">
              {/* Left: sidebar toggle */}
              <div className="flex items-center gap-2 min-w-0">
                {isMobile ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setMobileSidebarOpen(true)}
                    className="h-10 w-10 p-0 flex-shrink-0"
                    title="Open sidebar"
                  >
                    <Menu className="h-5 w-5" />
                  </Button>
                ) : (
                  !sidebarVisible && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setSidebarVisible(true)}
                      className="h-8 w-8 p-0"
                      title="Show sessions sidebar"
                    >
                      <PanelLeft className="h-4 w-4" />
                    </Button>
                  )
                )}
                {/* Branding — show when sidebar is hidden or on mobile */}
                {(isMobile || !sidebarVisible) && (
                  <Link href="/" className="flex items-end gap-2 min-w-0">
                    <span className="text-base md:text-lg font-bold truncate">
                      {isMobile ? "ACP" : "Ambient Code Platform"}
                    </span>
                    {!isMobile && version && (
                      <span className="text-[0.65rem] text-muted-foreground/60 pb-0.5">
                        {version}
                      </span>
                    )}
                  </Link>
                )}
              </div>

              {/* Right: nav items */}
              <div className="flex items-center gap-1 md:gap-3 flex-shrink-0">
                <RecentUpdatesDialog />
                <ThemeToggle />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => router.push('/integrations')}
                  className="text-muted-foreground hover:text-foreground hidden md:inline-flex"
                >
                  <Plug className="w-4 h-4 mr-1" />
                  Integrations
                </Button>
                {/* Mobile: icon-only integrations button */}
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => router.push('/integrations')}
                  className="text-muted-foreground hover:text-foreground h-10 w-10 p-0 md:hidden"
                  title="Integrations"
                >
                  <Plug className="w-4 h-4" />
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
