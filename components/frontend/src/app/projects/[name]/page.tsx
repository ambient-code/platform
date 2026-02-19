'use client';

import { useState, useEffect } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { Star, Settings, Users, Loader2, ChevronLeft, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { PageHeader } from '@/components/page-header';
import { Breadcrumbs } from '@/components/breadcrumbs';
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from "@/components/ui/resizable";

import { SessionsSection } from '@/components/workspace-sections/sessions-section';
import { SharingSection } from '@/components/workspace-sections/sharing-section';
import { SettingsSection } from '@/components/workspace-sections/settings-section';
import { useProject } from '@/services/queries/use-projects';

type Section = 'sessions' | 'sharing' | 'settings';

export default function ProjectDetailsPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const projectName = params?.name as string;

  // Fetch project data for display name and description
  const { data: project, isLoading: projectLoading } = useProject(projectName);

  // Initialize active section from query parameter or default to 'sessions'
  const initialSection = (searchParams.get('section') as Section) || 'sessions';
  const [activeSection, setActiveSection] = useState<Section>(initialSection);

  // Left panel visibility state (persisted to localStorage)
  const [leftPanelVisible, setLeftPanelVisible] = useState(() => {
    if (typeof window === 'undefined') return true;
    const saved = localStorage.getItem('workspace-left-panel-visible');
    return saved === null ? true : saved === 'true';
  });

  // Persist left panel visibility
  useEffect(() => {
    localStorage.setItem('workspace-left-panel-visible', String(leftPanelVisible));
  }, [leftPanelVisible]);

  // Update active section when query parameter changes
  useEffect(() => {
    const sectionParam = searchParams.get('section') as Section;
    if (sectionParam && ['sessions', 'sharing', 'settings'].includes(sectionParam)) {
      setActiveSection(sectionParam);
    }
  }, [searchParams]);

  const navItems = [
    { id: 'sessions' as Section, label: 'Sessions', icon: Star },
    { id: 'sharing' as Section, label: 'Sharing', icon: Users },
    { id: 'settings' as Section, label: 'Workspace Settings', icon: Settings },
  ];

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
    <div className="min-h-screen bg-background">
      {/* Sticky header */}
      <div className="sticky top-0 z-20 bg-card border-b">
        <div className="px-6 py-4">
          <Breadcrumbs
            items={[
              { label: 'Workspaces', href: '/projects' },
              { label: projectName },
            ]}
          />
        </div>
      </div>

      <div className="container mx-auto p-0">
        {/* Title and Description */}
        <div className="px-6 pt-6 pb-4">
          <PageHeader
            title={project?.displayName || projectName}
            description={project?.description || 'Manage agentic sessions, configure settings, and control access for this workspace'}
          />
        </div>

        {/* Divider */}
        <hr className="border-t mx-6 mb-6" />

        {/* Floating show button when left panel is hidden */}
        {!leftPanelVisible && (
          <Button
            variant="outline"
            size="sm"
            className="fixed left-2 top-1/2 -translate-y-1/2 z-30 h-8 w-8 p-0 rounded-full shadow-md"
            onClick={() => setLeftPanelVisible(true)}
            title="Show sidebar"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        )}

        {/* Content with resizable panels */}
        <div className="px-6">
          <ResizablePanelGroup
            direction="horizontal"
            autoSaveId="workspace-layout"
            className="min-h-[calc(100vh-16rem)]"
          >
            {leftPanelVisible && (
              <>
                <ResizablePanel
                  id="workspace-left-panel"
                  order={1}
                  defaultSize={20}
                  minSize={15}
                  maxSize={35}
                >
                  <div className="flex flex-col h-full pr-4">
                    {/* Sidebar Navigation */}
                    <Card className="flex-1">
                      <CardHeader>
                        <CardTitle>Workspace</CardTitle>
                      </CardHeader>
                      <CardContent className="px-4 pb-4 pt-2">
                        <div className="space-y-1">
                          {navItems.map((item) => {
                            const isActive = activeSection === item.id;
                            const Icon = item.icon;
                            return (
                              <Button
                                key={item.id}
                                variant={isActive ? "secondary" : "ghost"}
                                className={cn("w-full justify-start", isActive && "font-semibold")}
                                onClick={() => setActiveSection(item.id)}
                              >
                                <Icon className="w-4 h-4 mr-2" />
                                <span className="truncate">{item.label}</span>
                              </Button>
                            );
                          })}
                        </div>
                      </CardContent>
                    </Card>

                    {/* Hide panel button */}
                    <div className="pt-2 pb-3 flex justify-center">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setLeftPanelVisible(false)}
                        className="text-muted-foreground hover:text-foreground"
                      >
                        <ChevronLeft className="h-4 w-4 mr-1" />
                        <span className="text-xs">Hide sidebar</span>
                      </Button>
                    </div>
                  </div>
                </ResizablePanel>
                <ResizableHandle className="w-1 hover:bg-primary/50 transition-colors" />
              </>
            )}

            <ResizablePanel
              id="workspace-main-panel"
              order={2}
              defaultSize={leftPanelVisible ? 80 : 100}
              minSize={50}
            >
              <div className={cn(!leftPanelVisible && "pl-8")}>
                {/* Main Content */}
                {activeSection === 'sessions' && <SessionsSection projectName={projectName} />}
                {activeSection === 'sharing' && <SharingSection projectName={projectName} />}
                {activeSection === 'settings' && <SettingsSection projectName={projectName} />}
              </div>
            </ResizablePanel>
          </ResizablePanelGroup>
        </div>
      </div>
    </div>
  );
}
