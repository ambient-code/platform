"use client";

import { Gift, Loader2, ExternalLink } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { useGitHubReleases } from "@/services/queries/use-github-releases";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { useState } from "react";

export function RecentUpdatesDialog() {
  const [open, setOpen] = useState(false);
  const { data: releases, isLoading, isError, refetch } = useGitHubReleases();
  const [lastSeen, setLastSeen] = useLocalStorage<string | null>(
    "acp-last-seen-updates",
    null
  );

  const hasUnseen =
    releases &&
    releases.length > 0 &&
    (!lastSeen || new Date(releases[0].published_at) > new Date(lastSeen));

  const handleOpenChange = (isOpen: boolean) => {
    setOpen(isOpen);
    if (isOpen) {
      setLastSeen(new Date().toISOString());
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <Button
        variant="ghost"
        size="icon"
        className="h-9 w-9"
        aria-label="Recent updates"
        onClick={() => handleOpenChange(true)}
      >
        <div className="relative">
          <Gift className="h-[1.2rem] w-[1.2rem]" />
          {hasUnseen && (
            <span className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-red-500" />
          )}
        </div>
      </Button>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Recent Updates</DialogTitle>
        </DialogHeader>
        <div className="max-h-[60vh] overflow-y-auto space-y-6 pr-2">
          {isLoading && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          )}
          {isError && (
            <div className="flex flex-col items-center gap-3 py-8 text-center">
              <p className="text-sm text-muted-foreground">
                Failed to load updates.
              </p>
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                Retry
              </Button>
            </div>
          )}
          {!isLoading && !isError && releases?.length === 0 && (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No updates available.
            </p>
          )}
          {!isLoading &&
            !isError &&
            releases?.map((release) => (
              <div key={release.id} className="space-y-2">
                <div className="flex items-center gap-2 flex-wrap">
                  <h3 className="font-semibold">
                    {release.name || release.tag_name}
                  </h3>
                  <Badge variant="secondary">{release.tag_name}</Badge>
                  <span className="text-xs text-muted-foreground">
                    {formatDistanceToNow(new Date(release.published_at), {
                      addSuffix: true,
                    })}
                  </span>
                </div>
                {release.body && (
                  <div className="prose prose-sm dark:prose-invert max-w-none">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>
                      {release.body}
                    </ReactMarkdown>
                  </div>
                )}
                <a
                  href={release.html_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                >
                  View on GitHub
                  <ExternalLink className="h-3 w-3" />
                </a>
                <hr className="border-border" />
              </div>
            ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
