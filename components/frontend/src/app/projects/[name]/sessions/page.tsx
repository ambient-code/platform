'use client';

import { useParams } from 'next/navigation';
import { SessionsSection } from '@/components/workspace-sections/sessions-section';

export default function ProjectSessionsListPage() {
  const params = useParams();
  const projectName = params?.name as string;

  if (!projectName) return null;

  return (
    <div className="h-full overflow-auto p-6">
      <SessionsSection projectName={projectName} />
    </div>
  );
}
