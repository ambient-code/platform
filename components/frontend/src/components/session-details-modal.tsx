"use client";

import { useCallback, useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Download, Loader2 } from 'lucide-react';
import type { AgenticSession } from '@/types/agentic-session';
import { getPhaseColor } from '@/utils/session-helpers';
import { successToast } from '@/hooks/use-toast';
import { useSessionExport } from '@/services/queries/use-sessions';

function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  
  if (hours > 0) {
    const remainingMinutes = minutes % 60;
    const remainingSeconds = seconds % 60;
    return `${hours}h ${remainingMinutes}m ${remainingSeconds}s`;
  } else if (minutes > 0) {
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds}s`;
  } else {
    return `${seconds}s`;
  }
}

type SessionDetailsModalProps = {
  session: AgenticSession;
  projectName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  durationMs?: number;
  k8sResources?: {
    pvcName?: string;
    pvcSize?: string;
  };
  messageCount: number;
};

export function SessionDetailsModal({
  session,
  projectName,
  open,
  onOpenChange,
  durationMs,
  k8sResources,
  messageCount,
}: SessionDetailsModalProps) {
  const [exportingAgui, setExportingAgui] = useState(false);
  const [exportingLegacy, setExportingLegacy] = useState(false);
  const sessionName = session.metadata?.name || '';

  // Use React Query hook - only fetches when modal is open
  const { data: exportData, isLoading: loadingExport } = useSessionExport(
    projectName,
    sessionName,
    open // Only fetch when modal is open
  );

  const downloadFile = useCallback((data: unknown, filename: string) => {
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  }, []);

  const handleExportAgui = useCallback(() => {
    if (!exportData) return;
    setExportingAgui(true);
    try {
      downloadFile(exportData.aguiEvents, `${sessionName}-chat.json`);
      successToast('Chat exported successfully');
    } finally {
      setExportingAgui(false);
    }
  }, [exportData, sessionName, downloadFile]);

  const handleExportLegacy = useCallback(() => {
    if (!exportData?.legacyMessages) return;
    setExportingLegacy(true);
    try {
      downloadFile(exportData.legacyMessages, `${sessionName}-legacy-messages.json`);
      successToast('Legacy messages exported successfully');
    } finally {
      setExportingLegacy(false);
    }
  }, [exportData, sessionName, downloadFile]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px] max-h-[90vh] overflow-y-auto">
        <DialogHeader className="space-y-3">
          <DialogTitle>Session Details</DialogTitle>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-3">
            <div className="flex items-start gap-3">
              <span className="font-semibold text-foreground/80 min-w-[100px]">Status:</span>
              <Badge className={getPhaseColor(session.status?.phase || "Pending")}>
                {session.status?.phase || "Pending"}
              </Badge>
            </div>
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-foreground/80 min-w-[100px]">Model:</span>
              <span className="text-foreground">{session.spec.llmSettings.model}</span>
            </div>
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-foreground/80 min-w-[100px]">Temperature:</span>
              <span className="text-foreground">{session.spec.llmSettings.temperature}</span>
            </div>
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-foreground/80 min-w-[100px]">Mode:</span>
              <span className="text-foreground">{session.spec?.interactive ? "Interactive" : "Headless"}</span>
            </div>
            
            {/* startTime removed from simplified status */}
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-foreground/80 min-w-[100px]">Duration:</span>
              <span className="text-foreground">{typeof durationMs === "number" ? formatDuration(durationMs) : "-"}</span>
            </div>
            
            {k8sResources?.pvcName && (
              <div className="flex items-start gap-3">
                <span className="font-semibold text-foreground/80 min-w-[100px]">PVC:</span>
                <span className="text-foreground font-mono break-all">{k8sResources.pvcName}</span>
              </div>
            )}
            
            {k8sResources?.pvcSize && (
              <div className="flex items-start gap-3">
                <span className="font-semibold text-foreground/80 min-w-[100px]">PVC Size:</span>
                <span className="text-foreground">{k8sResources.pvcSize}</span>
              </div>
            )}
            
            {/* jobName removed from simplified status */}
            
            <div className="flex items-start gap-3">
              <span className="font-semibold text-foreground/80 min-w-[100px]">Messages:</span>
              <span className="text-foreground">{messageCount}</span>
            </div>
            
            {/* Export buttons */}
            <div className="pt-2 space-y-2">
              {loadingExport ? (
                <Button variant="outline" size="sm" disabled className="w-full">
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Loading...
                </Button>
              ) : (
                <>
                  <Button 
                    variant="outline" 
                    size="sm" 
                    onClick={handleExportAgui}
                    disabled={exportingAgui || !exportData}
                    className="w-full"
                  >
                    {exportingAgui ? (
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    ) : (
                      <Download className="w-4 h-4 mr-2" />
                    )}
                    {exportingAgui ? 'Exporting...' : 'Export Chat'}
                  </Button>
                  
                  {exportData?.hasLegacy && (
                    <Button 
                      variant="outline" 
                      size="sm" 
                      onClick={handleExportLegacy}
                      disabled={exportingLegacy}
                      className="w-full"
                    >
                      {exportingLegacy ? (
                        <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      ) : (
                        <Download className="w-4 h-4 mr-2" />
                      )}
                      {exportingLegacy ? 'Exporting...' : 'Export Legacy Messages'}
                    </Button>
                  )}
                </>
              )}
            </div>
          </div>
          
          {session.spec.initialPrompt && (
            <div className="pt-2">
              <div className="mb-2">
                <span className="font-semibold text-foreground/80">Session prompt:</span>
              </div>
              <div className="max-h-[200px] overflow-y-auto p-4 bg-muted/50 rounded-md border border-gray-200">
                <p className="whitespace-pre-wrap text-sm text-foreground leading-relaxed">{session.spec.initialPrompt}</p>
              </div>
            </div>
          )}

          {session.status?.conditions && session.status.conditions.length > 0 && (
            <div className="pt-4">
              <div className="text-xs uppercase tracking-wide text-gray-500 mb-2">Reconciliation Conditions</div>
              <div className="space-y-2">
                {session.status.conditions.map((condition, index) => (
                  <div key={`${condition.type}-${index}`} className="rounded border px-3 py-2 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold">{condition.type}</span>
                      <span className={`text-xs ${condition.status === "True" ? "text-green-600" : condition.status === "False" ? "text-red-600" : "text-yellow-600"}`}>
                        {condition.status}
                      </span>
                    </div>
                    <div className="text-xs text-gray-500">{condition.reason || "No reason provided"}</div>
                    {condition.message && (
                      <div className="text-sm text-gray-700 mt-1">{condition.message}</div>
                    )}
                    {condition.lastTransitionTime && (
                      <div className="text-xs text-gray-400 mt-1">Updated {new Date(condition.lastTransitionTime).toLocaleString()}</div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
