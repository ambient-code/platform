'use client';

import { useParams } from 'next/navigation';
import { SettingsSection } from '@/components/workspace-sections/settings-section';

export default function ProjectSettingsPage() {
  const params = useParams();
  const projectName = params?.name as string;

  if (!projectName) return null;

  return (
    <div className="h-full overflow-auto p-6">
      <SettingsSection projectName={projectName} />
    </div>
  );
}
