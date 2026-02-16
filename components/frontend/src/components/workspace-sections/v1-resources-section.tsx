'use client';

import { useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { RefreshCw, Search, ChevronLeft, ChevronRight, Users, Bot, Zap, ListTodo, GitBranch, Package } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { EmptyState } from '@/components/empty-state';

import {
  useV1Agents,
  useV1Skills,
  useV1Tasks,
  useV1Users,
  useV1Workflows,
} from '@/services/queries/v1';
import { useDebounce } from '@/hooks/use-debounce';

type V1ResourcesSectionProps = {
  projectName: string;
};

type ResourceTab = 'agents' | 'skills' | 'tasks' | 'users' | 'workflows';

type ResourceMeta = {
  key: ResourceTab;
  label: string;
  icon: LucideIcon;
  columns: string[];
  getRow: (item: Record<string, unknown>) => string[];
};

const RESOURCE_TABS: ResourceMeta[] = [
  {
    key: 'agents',
    label: 'Agents',
    icon: Bot,
    columns: ['Name', 'Repo URL', 'Created'],
    getRow: (item) => [
      String(item.name || '—'),
      String(item.repo_url || '—'),
      item.created_at ? formatDistanceToNow(new Date(String(item.created_at)), { addSuffix: true }) : '—',
    ],
  },
  {
    key: 'skills',
    label: 'Skills',
    icon: Zap,
    columns: ['Name', 'Repo URL', 'Created'],
    getRow: (item) => [
      String(item.name || '—'),
      String(item.repo_url || '—'),
      item.created_at ? formatDistanceToNow(new Date(String(item.created_at)), { addSuffix: true }) : '—',
    ],
  },
  {
    key: 'tasks',
    label: 'Tasks',
    icon: ListTodo,
    columns: ['Name', 'Repo URL', 'Created'],
    getRow: (item) => [
      String(item.name || '—'),
      String(item.repo_url || '—'),
      item.created_at ? formatDistanceToNow(new Date(String(item.created_at)), { addSuffix: true }) : '—',
    ],
  },
  {
    key: 'users',
    label: 'Users',
    icon: Users,
    columns: ['Name', 'Username', 'Created'],
    getRow: (item) => [
      String(item.name || '—'),
      String(item.username || '—'),
      item.created_at ? formatDistanceToNow(new Date(String(item.created_at)), { addSuffix: true }) : '—',
    ],
  },
  {
    key: 'workflows',
    label: 'Workflows',
    icon: GitBranch,
    columns: ['Name', 'Branch', 'Created'],
    getRow: (item) => [
      String(item.name || '—'),
      String(item.branch || '—'),
      item.created_at ? formatDistanceToNow(new Date(String(item.created_at)), { addSuffix: true }) : '—',
    ],
  },
];

export function V1ResourcesSection({ projectName }: V1ResourcesSectionProps) {
  const [activeTab, setActiveTab] = useState<ResourceTab>('agents');

  return (
    <Card className="flex-1">
      <CardHeader>
        <CardTitle>Resources (API Server)</CardTitle>
        <CardDescription>Browse resources from the ambient-api-server</CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as ResourceTab)}>
          <TabsList className="mb-4">
            {RESOURCE_TABS.map((tab) => {
              const Icon = tab.icon;
              return (
                <TabsTrigger key={tab.key} value={tab.key} className="gap-1.5">
                  <Icon className="h-3.5 w-3.5" />
                  {tab.label}
                </TabsTrigger>
              );
            })}
          </TabsList>

          {RESOURCE_TABS.map((tab) => (
            <TabsContent key={tab.key} value={tab.key}>
              <ResourceTabContent projectName={projectName} tab={tab} />
            </TabsContent>
          ))}
        </Tabs>
      </CardContent>
    </Card>
  );
}

function ResourceTabContent({ projectName, tab }: { projectName: string; tab: ResourceMeta }) {
  const [searchInput, setSearchInput] = useState('');
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const debouncedSearch = useDebounce(searchInput, 300);

  const opts = { page, size: pageSize, search: debouncedSearch || undefined };

  const agentsQuery = useV1Agents(projectName, opts);
  const skillsQuery = useV1Skills(projectName, opts);
  const tasksQuery = useV1Tasks(projectName, opts);
  const usersQuery = useV1Users(projectName, opts);
  const workflowsQuery = useV1Workflows(projectName, opts);

  const queryByTab = {
    agents: agentsQuery,
    skills: skillsQuery,
    tasks: tasksQuery,
    users: usersQuery,
    workflows: workflowsQuery,
  } as const;

  const activeQuery = queryByTab[tab.key];
  const rawData = activeQuery.data as { items?: Record<string, unknown>[]; total?: number } | undefined;
  const items = rawData?.items ?? [];
  const total = rawData?.total ?? 0;
  const totalPages = Math.ceil(total / pageSize);
  const isFetching = activeQuery.isFetching;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={`Search ${tab.label.toLowerCase()}...`}
            value={searchInput}
            onChange={(e) => { setSearchInput(e.target.value); setPage(1); }}
            className="pl-9"
          />
        </div>
        <Button variant="outline" size="sm" onClick={() => activeQuery.refetch()} disabled={isFetching}>
          <RefreshCw className={`w-4 h-4 mr-2 ${isFetching ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>

      {items.length === 0 ? (
        <EmptyState
          icon={Package}
          title={debouncedSearch ? `No matching ${tab.label.toLowerCase()}` : `No ${tab.label.toLowerCase()} found`}
          description={debouncedSearch ? `No results for "${debouncedSearch}"` : `No ${tab.label.toLowerCase()} exist on the API server yet`}
        />
      ) : (
        <>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  {tab.columns.map((col) => (
                    <TableHead key={col}>{col}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((item) => {
                  const cells = tab.getRow(item);
                  return (
                    <TableRow key={String(item.id)}>
                      {cells.map((cell, i) => (
                        <TableCell key={i} className={i === 0 ? 'font-medium' : ''}>
                          {cell}
                        </TableCell>
                      ))}
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-4 border-t">
              <div className="text-sm text-muted-foreground">
                Page {page} of {totalPages} ({total} total)
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1 || isFetching}
                >
                  <ChevronLeft className="h-4 w-4 mr-1" />
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => p + 1)}
                  disabled={page >= totalPages || isFetching}
                >
                  Next
                  <ChevronRight className="h-4 w-4 ml-1" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
