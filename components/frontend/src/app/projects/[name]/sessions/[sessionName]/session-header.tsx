"use client";

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { RefreshCw, Octagon, Trash2, Copy, MoreVertical, Info, Play, Pencil, Download, FileText, Printer, Loader2, HardDrive, Clock } from 'lucide-react';
import { CloneSessionDialog } from '@/components/clone-session-dialog';
import { SessionDetailsModal } from '@/components/session-details-modal';
import { EditSessionNameDialog } from '@/components/edit-session-name-dialog';
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
  DropdownMenuSeparator, DropdownMenuSub, DropdownMenuSubTrigger, DropdownMenuSubContent,
} from '@/components/ui/dropdown-menu';
import type { AgenticSession } from '@/types/agentic-session';
import { useUpdateSessionDisplayName, useCurrentUser, useSessionExport } from '@/services/queries';
import { useMcpStatus } from '@/services/queries/use-mcp';
import { useGoogleStatus } from '@/services/queries/use-google';
import { successToast, errorToast } from '@/hooks/use-toast';
import { saveToGoogleDrive } from '@/services/api/sessions';
import { convertEventsToMarkdown, downloadAsMarkdown, exportAsPdf } from '@/utils/export-chat';

type SessionHeaderProps = {
  session: AgenticSession;
  projectName: string;
  actionLoading: string | null;
  onRefresh: () => void;
  onStop: () => void;
  onContinue: () => void;
  onDelete: () => void;
  renderMode?: 'full' | 'actions-only' | 'kebab-only';
};

export function SessionHeader({
  session,
  projectName,
  actionLoading,
  onRefresh,
  onStop,
  onContinue,
  onDelete,
  renderMode = 'full',
}: SessionHeaderProps) {
  const [detailsModalOpen, setDetailsModalOpen] = useState(false);
  const [editNameDialogOpen, setEditNameDialogOpen] = useState(false);
  const [exportLoading, setExportLoading] = useState<'markdown' | 'pdf' | 'gdrive' | null>(null);

  const updateDisplayNameMutation = useUpdateSessionDisplayName();
  const { data: me } = useCurrentUser();

  const phase = session.status?.phase || "Pending";
  const isRunning = phase === "Running";
  const canStop = isRunning || phase === "Creating";
  const canResume = phase === "Stopped";
  const canDelete = phase === "Completed" || phase === "Failed" || phase === "Stopped";
  const stoppedDueToInactivity = phase === "Stopped" && session.status?.stoppedReason === "inactivity";

  const { refetch: fetchExportData } = useSessionExport(projectName, session.metadata.name, false);
  const { data: mcpStatus } = useMcpStatus(projectName, session.metadata.name, isRunning);
  const { data: googleStatus } = useGoogleStatus();
  const googleDriveServer = mcpStatus?.servers?.find(
    (s) => s.name.includes('gdrive') || s.name.includes('google-drive') || s.name.includes('google-workspace')
  );
  const hasGdriveMcp = !!googleDriveServer;

  const handleEditName = (newName: string) => {
    updateDisplayNameMutation.mutate(
      {
        projectName,
        sessionName: session.metadata.name,
        displayName: newName,
      },
      {
        onSuccess: () => {
          successToast('Session name updated successfully');
          setEditNameDialogOpen(false);
          onRefresh();
        },
        onError: (error) => {
          errorToast(error instanceof Error ? error.message : 'Failed to update session name');
        },
      }
    );
  };

  const handleExport = async (format: 'markdown' | 'pdf' | 'gdrive') => {
    if (format === 'gdrive') {
      if (!googleStatus?.connected) {
        errorToast('Connect Google Drive in Integrations first');
        return;
      }
      if (!isRunning || !hasGdriveMcp) {
        errorToast('Session must be running with Google Drive MCP configured');
        return;
      }
    }

    setExportLoading(format);
    try {
      const { data: exportData } = await fetchExportData();
      if (!exportData) {
        throw new Error('No export data available');
      }
      const markdown = convertEventsToMarkdown(exportData, session, {
        username: me?.displayName || me?.username || me?.email,
        projectName,
      });
      const filename = session.spec.displayName || session.metadata.name;

      switch (format) {
        case 'markdown':
          downloadAsMarkdown(markdown, `${filename}.md`);
          successToast('Chat exported as Markdown');
          break;
        case 'pdf':
          exportAsPdf(markdown, filename);
          break;
        case 'gdrive': {
          const result = await saveToGoogleDrive(
            projectName, session.metadata.name, markdown,
            `${filename}.md`, me?.email ?? '', googleDriveServer?.name ?? 'google-workspace',
          );
          if (result.error) {
            throw new Error(result.error);
          }
          if (!result.content) {
            throw new Error('Failed to create file in Google Drive');
          }
          successToast('Saved to Google Drive');
          break;
        }
      }
    } catch (err) {
      errorToast(err instanceof Error ? err.message : 'Failed to export chat');
    } finally {
      setExportLoading(null);
    }
  };

  const exportSubMenu = (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>
        <Download className="w-4 h-4 mr-2" />
        Export chat
      </DropdownMenuSubTrigger>
      <DropdownMenuSubContent>
        <DropdownMenuItem
          onClick={() => void handleExport('markdown')}
          disabled={exportLoading !== null}
        >
          {exportLoading === 'markdown' ? (
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
          ) : (
            <FileText className="w-4 h-4 mr-2" />
          )}
          As Markdown
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => void handleExport('pdf')}
          disabled={exportLoading !== null}
        >
          {exportLoading === 'pdf' ? (
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
          ) : (
            <Printer className="w-4 h-4 mr-2" />
          )}
          As PDF
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => void handleExport('gdrive')}
          disabled={exportLoading !== null}
        >
          {exportLoading === 'gdrive' ? (
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
          ) : (
            <HardDrive className="w-4 h-4 mr-2" />
          )}
          Save to my Google Drive
        </DropdownMenuItem>
      </DropdownMenuSubContent>
    </DropdownMenuSub>
  );

  // Kebab menu only (for breadcrumb line)
  if (renderMode === 'kebab-only') {
    return (
      <>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm">
              <MoreVertical className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onRefresh} disabled={actionLoading !== null}>
              <RefreshCw className="w-4 h-4 mr-2" />
              Refresh
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => setDetailsModalOpen(true)}>
              <Info className="w-4 h-4 mr-2" />
              View details
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setEditNameDialogOpen(true)}>
              <Pencil className="w-4 h-4 mr-2" />
              Edit name
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            {canStop && (
              <DropdownMenuItem
                onClick={onStop}
                disabled={actionLoading === "stopping"}
              >
                <Octagon className="w-4 h-4 mr-2" />
                {actionLoading === "stopping" ? "Stopping..." : "Stop"}
              </DropdownMenuItem>
            )}
            {canResume && (
              <DropdownMenuItem
                onClick={onContinue}
                disabled={actionLoading === "resuming"}
              >
                <Play className="w-4 h-4 mr-2" />
                {actionLoading === "resuming" ? "Resuming..." : "Resume"}
              </DropdownMenuItem>
            )}
            {(canStop || canResume) && <DropdownMenuSeparator />}
            <CloneSessionDialog
              session={session}
              trigger={
                <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
                  <Copy className="w-4 h-4 mr-2" />
                  Clone
                </DropdownMenuItem>
              }
              projectName={projectName}
            />
            {exportSubMenu}
            {canDelete && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={onDelete}
                  disabled={actionLoading === "deleting"}
                  className="text-red-600 dark:text-red-400"
                >
                  <Trash2 className="w-4 h-4 mr-2" />
                  {actionLoading === "deleting" ? "Deleting..." : "Delete"}
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>

        <SessionDetailsModal
          session={session}
          projectName={projectName}
          open={detailsModalOpen}
          onOpenChange={setDetailsModalOpen}
        />

        <EditSessionNameDialog
          open={editNameDialogOpen}
          onOpenChange={setEditNameDialogOpen}
          currentName={session.spec.displayName || session.metadata.name}
          onSave={handleEditName}
          isLoading={updateDisplayNameMutation.isPending}
        />
      </>
    );
  }

  // Actions only (Stop/Resume buttons) - for below breadcrumb
  if (renderMode === 'actions-only') {
    return (
      <div className="space-y-2">
        {stoppedDueToInactivity && (
          <Alert variant="info">
            <Clock className="h-4 w-4" />
            <AlertDescription>
              This session was automatically stopped after being idle. You can resume it to continue working.
            </AlertDescription>
          </Alert>
        )}
        <div className="flex items-start justify-start">
          <div className="flex gap-2">
            {canStop && (
              <Button
                variant="outline"
                size="sm"
                onClick={onStop}
                disabled={actionLoading === "stopping"}
                className="hover:border-red-600 hover:bg-red-50 group"
              >
                <Octagon className="w-4 h-4 mr-2 fill-red-200 stroke-red-500 group-hover:fill-red-500 group-hover:stroke-red-700 transition-colors" />
                Stop
              </Button>
            )}
            {canResume && (
              <Button
                variant="outline"
                size="sm"
                onClick={onContinue}
                disabled={actionLoading === "resuming"}
                className="hover:border-green-600 hover:bg-green-50 group"
              >
                <Play className="w-4 h-4 mr-2 fill-green-200 stroke-green-600 group-hover:fill-green-500 group-hover:stroke-green-700 transition-colors" />
                Resume
              </Button>
            )}
          </div>
        </div>
      </div>
    );
  }

  // Full mode (original layout)
  return (
    <div className="space-y-2">
      {stoppedDueToInactivity && (
        <Alert variant="info">
          <Clock className="h-4 w-4" />
          <AlertDescription>
            This session was automatically stopped after being idle. You can resume it to continue working.
          </AlertDescription>
        </Alert>
      )}
      <div className="flex items-start justify-end">
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onRefresh}
            disabled={actionLoading === "refreshing"}
          >
            <RefreshCw className={`w-4 h-4 mr-2 ${actionLoading === "refreshing" ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          {canStop && (
            <Button
              variant="outline"
              size="sm"
              onClick={onStop}
              disabled={actionLoading === "stopping"}
              className="hover:border-red-600 hover:bg-red-50 group"
            >
              <Octagon className="w-4 h-4 mr-2 fill-red-200 stroke-red-500 group-hover:fill-red-500 group-hover:stroke-red-700 transition-colors" />
              Stop
            </Button>
          )}
          {canResume && (
            <Button
              variant="outline"
              size="sm"
              onClick={onContinue}
              disabled={actionLoading === "resuming"}
              className="hover:border-green-600 hover:bg-green-50 group"
            >
              <Play className="w-4 h-4 mr-2 fill-green-200 stroke-green-600 group-hover:fill-green-500 group-hover:stroke-green-700 transition-colors" />
              Resume
            </Button>
          )}

          {/* Actions dropdown menu */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm">
                <MoreVertical className="w-4 h-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => setDetailsModalOpen(true)}>
                <Info className="w-4 h-4 mr-2" />
                View details
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setEditNameDialogOpen(true)}>
                <Pencil className="w-4 h-4 mr-2" />
                Edit name
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <CloneSessionDialog
                session={session}
                trigger={
                  <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
                    <Copy className="w-4 h-4 mr-2" />
                    Clone
                  </DropdownMenuItem>
                }
                projectName={projectName}
              />
              {exportSubMenu}
              {canDelete && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={onDelete}
                    disabled={actionLoading === "deleting"}
                    className="text-red-600 dark:text-red-400"
                  >
                    <Trash2 className="w-4 h-4 mr-2" />
                    {actionLoading === "deleting" ? "Deleting..." : "Delete"}
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <SessionDetailsModal
        session={session}
        projectName={projectName}
        open={detailsModalOpen}
        onOpenChange={setDetailsModalOpen}
      />

      <EditSessionNameDialog
        open={editNameDialogOpen}
        onOpenChange={setEditNameDialogOpen}
        currentName={session.spec.displayName || session.metadata.name}
        onSave={handleEditName}
        isLoading={updateDisplayNameMutation.isPending}
      />
    </div>
  );
}
