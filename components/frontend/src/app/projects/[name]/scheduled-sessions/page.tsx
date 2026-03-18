'use client';

import { useParams } from 'next/navigation';
import { SchedulesSection } from '@/components/workspace-sections/scheduled-sessions-tab';

export default function ScheduledSessionsPage() {
  const params = useParams();
  const projectName = params?.name as string;

  if (!projectName) return null;

  return (
    <div className="h-full overflow-auto p-6">
      <SchedulesSection projectName={projectName} />
    </div>
  );
}
