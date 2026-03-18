"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { UserBubble } from "@/components/user-bubble";
import { ThemeToggle } from "@/components/theme-toggle";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetClose } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Plug, LogOut, Menu, Home, MessageSquare } from "lucide-react";
import { useVersion } from "@/services/queries/use-version";
import { useIsMobile } from "@/hooks/use-mobile";

type NavigationProps = {
  feedbackUrl?: string;
};

export function Navigation({ feedbackUrl }: NavigationProps) {
  // const pathname = usePathname();
  // const segments = pathname?.split("/").filter(Boolean) || [];
  const router = useRouter();
  const { data: version } = useVersion();
  const isMobile = useIsMobile();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleLogout = () => {
    // Redirect to oauth-proxy logout endpoint
    // This clears the OpenShift OAuth session and redirects back to login
    window.location.href = '/oauth/sign_out';
  };

  const handleMobileNav = (path: string) => {
    setMobileMenuOpen(false);
    router.push(path);
  };

  return (
    <nav className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="px-6">
        <div className="flex h-16 items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            {/* Mobile hamburger menu button */}
            {isMobile && (
              <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
                <Button
                  variant="ghost"
                  size="icon"
                  className="md:hidden h-9 w-9"
                  onClick={() => setMobileMenuOpen(true)}
                  aria-label="Open navigation menu"
                >
                  <Menu className="h-5 w-5" />
                </Button>
                <SheetContent side="left" className="w-72">
                  <SheetHeader>
                    <SheetTitle>
                      <span className="text-lg font-bold">ACP</span>
                      {version && (
                        <span className="ml-2 text-xs text-muted-foreground">{version}</span>
                      )}
                    </SheetTitle>
                  </SheetHeader>
                  <div className="flex flex-col gap-1 px-4">
                    <SheetClose
                      onClick={() => handleMobileNav('/')}
                      className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground transition-colors text-left"
                    >
                      <Home className="h-4 w-4" />
                      Home
                    </SheetClose>
                    <SheetClose
                      onClick={() => handleMobileNav('/integrations')}
                      className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground transition-colors text-left"
                    >
                      <Plug className="h-4 w-4" />
                      Integrations
                    </SheetClose>
                    {feedbackUrl && (
                      <a
                        href={feedbackUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        onClick={() => setMobileMenuOpen(false)}
                        className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground transition-colors"
                      >
                        <MessageSquare className="h-4 w-4" />
                        Share feedback
                      </a>
                    )}
                  </div>
                  <div className="mt-auto border-t px-4 py-4">
                    <button
                      onClick={() => {
                        setMobileMenuOpen(false);
                        handleLogout();
                      }}
                      className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors"
                    >
                      <LogOut className="h-4 w-4" />
                      Logout
                    </button>
                  </div>
                </SheetContent>
              </Sheet>
            )}
            <div className="flex items-end gap-2">
              <Link href="/" className="text-lg font-bold">
                <span className="hidden md:inline">Ambient Code Platform</span>
                <span className="md:hidden">ACP</span>
              </Link>
              {version && (
                <a
                  href="https://github.com/ambient-code/platform/releases"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[0.65rem] text-muted-foreground/60 pb-0.75 hover:text-muted-foreground transition-colors"
                >
                  <span>{version}</span>
                </a>
              )}
            </div>
          </div>
          {/* Desktop navigation items - hidden on mobile */}
          <div className="hidden md:flex items-center gap-3">
            {feedbackUrl && (
              <a
                href={feedbackUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Share feedback
              </a>
            )}
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
          {/* Mobile: only show theme toggle (user menu items are in the drawer) */}
          <div className="flex md:hidden items-center gap-2">
            <ThemeToggle />
          </div>
        </div>
      </div>
    </nav>
  );
}
