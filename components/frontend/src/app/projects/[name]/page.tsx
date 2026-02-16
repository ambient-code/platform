'use client';

import { useState, useEffect } from 'react';
import { useParams, useSearchParams } from 'next/navigation';
import { Star, Settings, Users, Loader2, Package, Shield, GitFork, KeyRound } from 'lucide-react';
import { cn } from '@/lib/utils';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { PageHeader } from '@/components/page-header';
import { Breadcrumbs } from '@/components/breadcrumbs';

import { SessionsSection } from '@/components/workspace-sections/sessions-section';
import { SharingSection } from '@/components/workspace-sections/sharing-section';
import { SettingsSection } from '@/components/workspace-sections/settings-section';
import { V1SessionsSection } from '@/components/workspace-sections/v1-sessions-section';
import { V1ResourcesSection } from '@/components/workspace-sections/v1-resources-section';
import { V1PermissionsSection } from '@/components/workspace-sections/v1-permissions-section';
import { V1RepositoryRefsSection } from '@/components/workspace-sections/v1-repository-refs-section';
import { V1ProjectKeysSection } from '@/components/workspace-sections/v1-project-keys-section';
import { useProject } from '@/services/queries/use-projects';
import { useApiSource } from '@/contexts/api-source-context';

type Section = 'sessions' | 'sharing' | 'settings' | 'v1-sessions' | 'v1-resources' | 'v1-permissions' | 'v1-repositories' | 'v1-api-keys';

export default function ProjectDetailsPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const projectName = params?.name as string;
  
  const { source, isApiServer } = useApiSource();

  const { data: project, isLoading: projectLoading } = useProject(projectName, { enabled: !isApiServer });

  const initialSection = (searchParams.get('section') as Section) || (isApiServer ? 'v1-sessions' : 'sessions');
  const [activeSection, setActiveSection] = useState<Section>(initialSection);

  useEffect(() => {
    const sectionParam = searchParams.get('section') as Section;
    if (sectionParam && ['sessions', 'sharing', 'settings', 'v1-sessions', 'v1-resources', 'v1-permissions', 'v1-repositories', 'v1-api-keys'].includes(sectionParam)) {
      setActiveSection(sectionParam);
    }
  }, [searchParams]);

  useEffect(() => {
    const k8sSections: Section[] = ['sessions', 'sharing', 'settings'];
    const apiSections: Section[] = ['v1-sessions', 'v1-resources', 'v1-permissions', 'v1-repositories', 'v1-api-keys'];
    if (source === 'k8s' && !k8sSections.includes(activeSection)) {
      setActiveSection('sessions');
    } else if (source === 'api-server' && !apiSections.includes(activeSection)) {
      setActiveSection('v1-sessions');
    }
  }, [source, activeSection]);

  const k8sNavItems = [
    { id: 'sessions' as Section, label: 'Sessions', icon: Star },
    { id: 'sharing' as Section, label: 'Sharing', icon: Users },
    { id: 'settings' as Section, label: 'Workspace Settings', icon: Settings },
  ];

  const apiServerNavItems = [
    { id: 'v1-sessions' as Section, label: 'Sessions', icon: Star },
    { id: 'v1-resources' as Section, label: 'Resources', icon: Package },
    { id: 'v1-permissions' as Section, label: 'Permissions', icon: Shield },
    { id: 'v1-repositories' as Section, label: 'Repositories', icon: GitFork },
    { id: 'v1-api-keys' as Section, label: 'API Keys', icon: KeyRound },
  ];

  // Loading state
  if (!projectName || (!isApiServer && projectLoading)) {
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

        {/* Content */}
        <div className="px-6 flex gap-6">
          <aside className="w-56 shrink-0 space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>{source === 'k8s' ? 'Workspace' : 'API Server'}</CardTitle>
              </CardHeader>
              <CardContent className="px-4 pb-4 pt-2">
                <div className="space-y-1">
                  {(source === 'k8s' ? k8sNavItems : apiServerNavItems).map((item) => {
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
                        {item.label}
                      </Button>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          </aside>

          {activeSection === 'sessions' && <SessionsSection projectName={projectName} />}
          {activeSection === 'sharing' && <SharingSection projectName={projectName} />}
          {activeSection === 'settings' && <SettingsSection projectName={projectName} />}
          {activeSection === 'v1-sessions' && <V1SessionsSection projectName={projectName} />}
          {activeSection === 'v1-resources' && <V1ResourcesSection projectName={projectName} />}
          {activeSection === 'v1-permissions' && <V1PermissionsSection projectName={projectName} />}
          {activeSection === 'v1-repositories' && <V1RepositoryRefsSection projectName={projectName} />}
          {activeSection === 'v1-api-keys' && <V1ProjectKeysSection projectName={projectName} />}
        </div>
      </div>
    </div>
  );
}
