'use client';

import { useState, useEffect } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { Star, Settings, Users, KeyRound, Loader2 } from 'lucide-react';

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { PageHeader } from '@/components/page-header';
import Link from 'next/link';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import {
  SidebarProvider,
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarRail,
  SidebarInset,
  SidebarTrigger,
  useSidebar,
} from '@/components/ui/sidebar';
import { Separator } from '@/components/ui/separator';

import { SessionsSection } from '@/components/workspace-sections/sessions-section';
import { SharingSection } from '@/components/workspace-sections/sharing-section';
import { SettingsSection } from '@/components/workspace-sections/settings-section';
import { KeysSection } from '@/components/workspace-sections/keys-section';
import { useProject } from '@/services/queries/use-projects';

type Section = 'sessions' | 'sharing' | 'keys' | 'settings';

const navItems: { id: Section; label: string; icon: typeof Star }[] = [
  { id: 'sessions', label: 'Sessions', icon: Star },
  { id: 'sharing', label: 'Sharing', icon: Users },
  { id: 'keys', label: 'Access Keys', icon: KeyRound },
  { id: 'settings', label: 'Workspace Settings', icon: Settings },
];

function WorkspaceSidebar({
  activeSection,
  onSectionChange,
}: {
  activeSection: Section;
  onSectionChange: (section: Section) => void;
}) {
  const { isMobile, setOpenMobile } = useSidebar();

  return (
    <Sidebar collapsible="offcanvas" data-below-nav="true">
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Workspace</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map((item) => (
                <SidebarMenuItem key={item.id}>
                  <SidebarMenuButton
                    isActive={activeSection === item.id}
                    onClick={() => {
                      onSectionChange(item.id);
                      if (isMobile) setOpenMobile(false);
                    }}
                    tooltip={item.label}
                  >
                    <item.icon />
                    <span>{item.label}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
}

export default function ProjectDetailsPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const projectName = params?.name as string;

  // Fetch project data for display name and description
  const { data: project, isLoading: projectLoading } = useProject(projectName);

  // Initialize active section from query parameter or default to 'sessions'
  const initialSection = (searchParams.get('section') as Section) || 'sessions';
  const [activeSection, setActiveSection] = useState<Section>(initialSection);

  // Update active section when query parameter changes
  useEffect(() => {
    const sectionParam = searchParams.get('section') as Section;
    if (sectionParam && ['sessions', 'sharing', 'keys', 'settings'].includes(sectionParam)) {
      setActiveSection(sectionParam);
    }
  }, [searchParams]);

  // Loading state
  if (!projectName || projectLoading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <Alert className="max-w-md mx-4">
            <Loader2 className="h-4 w-4 animate-spin" />
            <AlertTitle>Loading Workspace...</AlertTitle>
            <AlertDescription>
              <p>Please wait while the workspace is loading...</p>
            </AlertDescription>
          </Alert>
        </div>
      </div>
    );
  }

  return (
    <SidebarProvider
      defaultOpen={true}
      className="min-h-[calc(100svh-4rem)]"
    >
      <WorkspaceSidebar
        activeSection={activeSection}
        onSectionChange={setActiveSection}
      />
      <SidebarInset>
        {/* Sticky header with breadcrumbs and sidebar trigger */}
        <header className="sticky top-0 z-20 flex items-center gap-2 bg-background border-b px-4 h-12">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="mr-2 h-4" />
          <Breadcrumb>
            <BreadcrumbList>
              <BreadcrumbItem>
                <BreadcrumbLink asChild>
                  <Link href="/projects">Workspaces</Link>
                </BreadcrumbLink>
              </BreadcrumbItem>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                <BreadcrumbPage>{projectName}</BreadcrumbPage>
              </BreadcrumbItem>
            </BreadcrumbList>
          </Breadcrumb>
        </header>

        {/* Page content */}
        <div className="p-6">
          <PageHeader
            title={project?.displayName || projectName}
            description={project?.description || 'Manage agentic sessions, configure settings, and control access for this workspace'}
          />

          <hr className="border-t my-6" />

          {/* Main Content */}
          {activeSection === 'sessions' && <SessionsSection projectName={projectName} />}
          {activeSection === 'sharing' && <SharingSection projectName={projectName} />}
          {activeSection === 'keys' && <KeysSection projectName={projectName} />}
          {activeSection === 'settings' && <SettingsSection projectName={projectName} />}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
