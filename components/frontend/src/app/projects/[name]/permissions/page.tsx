'use client';

import { useParams } from 'next/navigation';
import { SharingSection } from '@/components/workspace-sections/sharing-section';

export default function PermissionsPage() {
  const params = useParams();
  const projectName = params?.name as string;

  if (!projectName) return null;

  return (
    <div className="h-full overflow-auto p-6">
      <SharingSection projectName={projectName} />
    </div>
  );
}
