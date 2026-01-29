"use client";

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { RefreshCw, Octagon, Trash2, Copy, MoreVertical, Info, Play, Pencil } from 'lucide-react';
import { CloneSessionDialog } from '@/components/clone-session-dialog';
import { SessionDetailsModal } from '@/components/session-details-modal';
import { EditSessionNameDialog } from '@/components/edit-session-name-dialog';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger, DropdownMenuSeparator } from '@/components/ui/dropdown-menu';
import type { AgenticSession } from '@/types/agentic-session';
import { useUpdateSessionDisplayName } from '@/services/queries';
import { successToast, errorToast } from '@/hooks/use-toast';

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
  
  const updateDisplayNameMutation = useUpdateSessionDisplayName();
  
  const phase = session.status?.phase || "Pending";
  const canStop = phase === "Running" || phase === "Creating";
  const canResume = phase === "Stopped";
  const canDelete = phase === "Completed" || phase === "Failed" || phase === "Stopped";
  
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
      <div>
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
    <div>
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
