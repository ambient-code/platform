'use client';

import { useParams } from 'next/navigation';
import { ProjectMemorySection } from '@/components/workspace-sections/project-memory-section';
import { useWorkspaceFlag } from '@/services/queries/use-feature-flags-admin';

export default function ProjectMemoryPage() {
  const params = useParams();
  const projectName = params?.name as string;
  const { enabled: memoryEnabled, isLoading: flagLoading } = useWorkspaceFlag(
    projectName,
    'learning-agent-loop'
  );

  if (!projectName) return null;

  if (flagLoading) {
    return (
      <div className="h-full overflow-auto p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-8 w-48 bg-muted rounded" />
          <div className="h-64 w-full bg-muted rounded" />
        </div>
      </div>
    );
  }

  if (!memoryEnabled) {
    return (
      <div className="h-full overflow-auto p-6">
        <div className="text-center py-16">
          <h2 className="text-lg font-semibold mb-2">Project Memory</h2>
          <p className="text-sm text-muted-foreground">
            This feature is not enabled for this workspace. Enable the{" "}
            <code className="text-xs bg-muted px-1 py-0.5 rounded">
              learning-agent-loop
            </code>{" "}
            feature flag in Workspace Settings to use Project Memory.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto p-6">
      <ProjectMemorySection projectName={projectName} />
    </div>
  );
}
