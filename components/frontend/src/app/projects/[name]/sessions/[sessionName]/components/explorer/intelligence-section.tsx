"use client";

import { useState } from "react";
import {
  Brain,
  ChevronRight,
  Code2,
  Wrench,
  TestTube2,
  Layers,
  ScrollText,
  AlertTriangle,
  Clock,
  Shield,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { useRepoIntelligence } from "@/services/queries/use-intelligence";
import type { RepoIntelligence } from "@/services/api/intelligence";

type IntelligenceSectionProps = {
  projectName: string;
  repoUrl: string;
};

function IntelligenceField({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ElementType;
  label: string;
  value: string | undefined | null;
}) {
  if (!value) return null;
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground uppercase tracking-wider">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      <p className="text-sm leading-relaxed whitespace-pre-wrap">{value}</p>
    </div>
  );
}

function IntelligenceSheet({
  intel,
  open,
  onOpenChange,
}: {
  intel: RepoIntelligence;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const repoName = intel.repo_url.split("/").pop()?.replace(".git", "") || intel.repo_url;
  const analyzedDate = intel.analyzed_at
    ? new Date(intel.analyzed_at).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    : intel.created_at
      ? new Date(intel.created_at).toLocaleDateString("en-US", {
          month: "short",
          day: "numeric",
          year: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        })
      : null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg overflow-y-auto">
        <SheetHeader className="pb-2">
          <div className="flex items-center gap-2">
            <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-violet-500/10">
              <Brain className="h-4 w-4 text-violet-500" />
            </div>
            <div>
              <SheetTitle className="text-base">{repoName}</SheetTitle>
              <SheetDescription className="text-xs">
                Project Intelligence
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>

        {/* Meta badges */}
        <div className="flex flex-wrap gap-1.5 px-4 pb-3">
          <Badge variant="secondary" className="text-xs gap-1">
            <Code2 className="h-3 w-3" />
            {intel.language}
          </Badge>
          {intel.framework && (
            <Badge variant="outline" className="text-xs">
              {intel.framework.split(",")[0].trim()}
            </Badge>
          )}
          {intel.confidence != null && (
            <Badge
              variant="outline"
              className={`text-xs gap-1 ${
                intel.confidence >= 0.8
                  ? "border-green-300 dark:border-green-800 text-green-700 dark:text-green-400"
                  : intel.confidence >= 0.5
                    ? "border-yellow-300 dark:border-yellow-800 text-yellow-700 dark:text-yellow-400"
                    : "border-red-300 dark:border-red-800 text-red-700 dark:text-red-400"
              }`}
            >
              <Shield className="h-3 w-3" />
              {Math.round(intel.confidence * 100)}% confidence
            </Badge>
          )}
        </div>

        {/* Content sections */}
        <div className="flex-1 overflow-y-auto px-4 pb-4 space-y-5">
          {/* Summary — always first, full width */}
          {intel.summary && (
            <div className="rounded-lg bg-muted/50 p-3 space-y-1.5">
              <div className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Summary
              </div>
              <p className="text-sm leading-relaxed">{intel.summary}</p>
            </div>
          )}

          <div className="border-t pt-4 space-y-5">
            <IntelligenceField
              icon={Wrench}
              label="Build System"
              value={intel.build_system}
            />
            <IntelligenceField
              icon={TestTube2}
              label="Test Strategy"
              value={intel.test_strategy}
            />
            <IntelligenceField
              icon={Layers}
              label="Architecture"
              value={intel.architecture}
            />
            <IntelligenceField
              icon={ScrollText}
              label="Conventions"
              value={intel.conventions}
            />
            <IntelligenceField
              icon={AlertTriangle}
              label="Caveats"
              value={intel.caveats}
            />
          </div>

          {/* Footer metadata */}
          <div className="border-t pt-4 space-y-2 text-xs text-muted-foreground">
            {analyzedDate && (
              <div className="flex items-center gap-1.5">
                <Clock className="h-3 w-3" />
                Analyzed {analyzedDate}
              </div>
            )}
            {intel.analyzed_by_session_id && (
              <div className="flex items-center gap-1.5">
                <Brain className="h-3 w-3" />
                Session: <code className="text-[10px] bg-muted px-1 py-0.5 rounded">{intel.analyzed_by_session_id}</code>
              </div>
            )}
            <div className="flex items-center gap-1.5">
              v{intel.version} &middot; branch: {intel.repo_branch}
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}

export function IntelligenceSection({
  projectName,
  repoUrl,
}: IntelligenceSectionProps) {
  const [sheetOpen, setSheetOpen] = useState(false);
  const { data: intel, isLoading } = useRepoIntelligence(
    projectName,
    repoUrl
  );

  if (isLoading) {
    return (
      <div className="px-2 pb-2 pl-10">
        <div className="text-xs text-muted-foreground">
          Loading intelligence...
        </div>
      </div>
    );
  }

  if (!intel) {
    return null;
  }

  return (
    <>
      <div className="px-2 pb-2 pl-10 border-t border-dashed mt-1 pt-1.5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Brain className="h-3 w-3 text-violet-500" />
            <span className="font-medium text-foreground/80">Analyzed</span>
            <span className="text-muted-foreground/70">&middot;</span>
            <span>
              {intel.language}
              {intel.framework ? ` \u00b7 ${intel.framework.split(",")[0].trim()}` : ""}
            </span>
          </div>
          <button
            type="button"
            className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-0.5 cursor-pointer transition-colors"
            onClick={(e) => {
              e.stopPropagation();
              setSheetOpen(true);
            }}
          >
            <span>Details</span>
            <ChevronRight className="h-2.5 w-2.5" />
          </button>
        </div>
      </div>

      <IntelligenceSheet
        intel={intel}
        open={sheetOpen}
        onOpenChange={setSheetOpen}
      />
    </>
  );
}
