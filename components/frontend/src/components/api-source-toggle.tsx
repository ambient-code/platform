'use client';

import { Database, Server } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useApiSource } from '@/contexts/api-source-context';
import { cn } from '@/lib/utils';

export function ApiSourceToggle({ className }: { className?: string }) {
  const { source, setSource } = useApiSource();

  return (
    <Select value={source} onValueChange={(v) => setSource(v as 'k8s' | 'api-server')}>
      <SelectTrigger className={cn('w-[130px] h-8 text-xs', className)}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="k8s">
          <span className="flex items-center gap-1.5">
            <Database className="h-3 w-3" />
            Kubernetes
          </span>
        </SelectItem>
        <SelectItem value="api-server">
          <span className="flex items-center gap-1.5">
            <Server className="h-3 w-3" />
            API Server
          </span>
        </SelectItem>
      </SelectContent>
    </Select>
  );
}
