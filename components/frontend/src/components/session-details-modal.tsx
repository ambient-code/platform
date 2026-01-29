"use client";

import { useCallback, useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion';
import { Download, Loader2 } from 'lucide-react';
import type { AgenticSession } from '@/types/agentic-session';
import { getPhaseColor } from '@/utils/session-helpers';
import { successToast } from '@/hooks/use-toast';
import { useSessionExport } from '@/services/queries/use-sessions';

type SessionDetailsModalProps = {
  session: AgenticSession;
  projectName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function SessionDetailsModal({
  session,
  projectName,
  open,
  onOpenChange,
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
              <Accordion type="multiple" className="w-full">
                {session.status.conditions.map((condition, index) => (
                  <AccordionItem key={`${condition.type}-${index}`} value={`condition-${index}`}>
                    <AccordionTrigger className="py-3 px-3 hover:no-underline hover:bg-muted/50 rounded-t">
                      <div className="flex items-center justify-between flex-1 mr-2">
                        <span className="font-medium text-sm">{condition.type}</span>
                        <Badge 
                          variant={condition.status === "True" ? "default" : condition.status === "False" ? "destructive" : "secondary"}
                          className="ml-2"
                        >
                          {condition.status}
                        </Badge>
                      </div>
                    </AccordionTrigger>
                    <AccordionContent className="px-3 pb-3">
                      <div className="space-y-2 text-sm">
                        <div>
                          <span className="font-semibold text-foreground/70">Reason:</span>
                          <span className="ml-2 text-foreground/90">{condition.reason || "No reason provided"}</span>
                        </div>
                        {condition.message && (
                          <div>
                            <span className="font-semibold text-foreground/70">Message:</span>
                            <p className="mt-1 text-foreground/90">{condition.message}</p>
                          </div>
                        )}
                        {condition.lastTransitionTime && (
                          <div className="text-xs text-muted-foreground pt-2">
                            Updated {new Date(condition.lastTransitionTime).toLocaleString()}
                          </div>
                        )}
                      </div>
                    </AccordionContent>
                  </AccordionItem>
                ))}
              </Accordion>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
