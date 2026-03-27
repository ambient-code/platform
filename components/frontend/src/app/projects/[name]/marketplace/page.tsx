"use client";

import { useState, useMemo } from "react";
import { useParams } from "next/navigation";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Check, ExternalLink, Loader2, Plus, Search, Trash2 } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { toast } from "sonner";
import {
  useMarketplaceSources,
  useMarketplaceCatalog,
  useInstalledItems,
  useInstallItems,
  useUninstallItem,
} from "@/services/queries/use-marketplace";
import { ImportSourceDialog } from "@/components/import-source-dialog";
import type { MarketplaceCatalogItem, InstalledItem, MarketplaceSource } from "@/types/marketplace";
import { MARKETPLACE_CATEGORY_COLORS } from "@/types/marketplace";

type TypeFilter = "all" | "skill" | "command" | "agent";

function getSourceFileUrl(source: MarketplaceSource, filePath: string): string {
  const repoUrl = source.url.replace(/\.git$/, "");
  return `${repoUrl}/tree/${source.branch}/${filePath}`;
}

function SourceSection({
  source,
  sourceIndex,
  searchTerm,
  typeFilter,
  installedIds,
  onDirectInstall,
  onSelectItem,
  installingId,
}: {
  source: MarketplaceSource;
  sourceIndex: number;
  searchTerm: string;
  typeFilter: TypeFilter;
  installedIds: Set<string>;
  onDirectInstall: (item: MarketplaceCatalogItem, source: MarketplaceSource) => void;
  onSelectItem: (item: MarketplaceCatalogItem, source: MarketplaceSource) => void;
  installingId: string | null;
}) {
  const { data: items, isLoading } = useMarketplaceCatalog(sourceIndex);

  const filtered = useMemo(() => {
    if (!items) return [];
    return items.filter((item) => {
      if (typeFilter !== "all" && item.category !== typeFilter) return false;
      if (searchTerm) {
        const term = searchTerm.toLowerCase();
        return (
          item.name.toLowerCase().includes(term) ||
          item.description.toLowerCase().includes(term)
        );
      }
      return true;
    });
  }, [items, searchTerm, typeFilter]);

  if (isLoading) {
    return (
      <div className="space-y-3">
        <h3 className="text-lg font-semibold">{source.name}</h3>
        <div className="grid gap-2 grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full rounded-lg" />
          ))}
        </div>
      </div>
    );
  }

  if (filtered.length === 0 && items && items.length > 0) return null;
  if (!items || items.length === 0) return null;

  return (
    <div className="space-y-3">
      <div>
        <h3 className="text-lg font-semibold">{source.name}</h3>
        {source.description && (
          <p className="text-sm text-muted-foreground">{source.description}</p>
        )}
      </div>
      <div className="grid gap-2 grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
        {filtered.map((item) => {
          const isInstalled = installedIds.has(item.id);
          const isInstalling = installingId === item.id;
          return (
            <div
              key={item.id}
              className="flex items-start gap-2 p-2.5 rounded-lg border hover:bg-accent/50 cursor-pointer transition-colors"
              onClick={() => onSelectItem(item, source)}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <span className="text-sm font-medium truncate">{item.name}</span>
                  <Badge variant="secondary" className={`text-[10px] px-1.5 py-0 ${MARKETPLACE_CATEGORY_COLORS[item.category]}`}>
                    {item.category}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground line-clamp-1 mt-0.5">
                  {item.description}
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0"
                disabled={isInstalled || isInstalling}
                onClick={(e) => {
                  e.stopPropagation();
                  onDirectInstall(item, source);
                }}
              >
                {isInstalling ? (
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                ) : isInstalled ? (
                  <Check className="w-3.5 h-3.5 text-green-600" />
                ) : (
                  <Plus className="w-3.5 h-3.5" />
                )}
              </Button>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default function MarketplacePage() {
  const params = useParams();
  const projectName = params?.name as string;

  const [searchTerm, setSearchTerm] = useState("");
  const [typeFilter, setTypeFilter] = useState<TypeFilter>("all");
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [installingId, setInstallingId] = useState<string | null>(null);
  const [selectedItem, setSelectedItem] = useState<{ item: MarketplaceCatalogItem; source: MarketplaceSource } | null>(null);

  const { data: sources, isLoading: sourcesLoading } = useMarketplaceSources();
  const { data: installed, isLoading: installedLoading } = useInstalledItems(projectName);
  const installMutation = useInstallItems();
  const uninstallMutation = useUninstallItem();

  const installedIds = useMemo(() => {
    if (!installed) return new Set<string>();
    return new Set(installed.map((i) => i.itemId));
  }, [installed]);

  const filteredInstalled = useMemo(() => {
    if (!installed) return [];
    return installed.filter((item) => {
      if (typeFilter !== "all" && item.itemType !== typeFilter) return false;
      if (searchTerm) {
        const term = searchTerm.toLowerCase();
        return item.itemName.toLowerCase().includes(term);
      }
      return true;
    });
  }, [installed, typeFilter, searchTerm]);

  const handleDirectInstall = (item: MarketplaceCatalogItem, source: MarketplaceSource) => {
    setInstallingId(item.id);
    const installItem: InstalledItem = {
      sourceUrl: source.url,
      sourceBranch: source.branch,
      sourcePath: source.path,
      itemId: item.id,
      itemType: item.category,
      itemName: item.name,
      filePath: item.file_path,
    };
    installMutation.mutate(
      { projectName, items: [installItem] },
      {
        onSuccess: () => {
          toast.success(`Installed "${item.name}"`);
          setInstallingId(null);
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : "Install failed");
          setInstallingId(null);
        },
      }
    );
  };

  const handleUninstall = (itemId: string, itemName: string) => {
    uninstallMutation.mutate(
      { projectName, itemId },
      {
        onSuccess: () => toast.success(`Removed "${itemName}"`),
        onError: (error) =>
          toast.error(error instanceof Error ? error.message : "Remove failed"),
      }
    );
  };

  if (!projectName) return null;

  return (
    <div className="h-full overflow-auto p-6">
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Marketplace</h1>
          <p className="text-muted-foreground">
            Browse and import skills, commands, and agents for your workspace
          </p>
        </div>

        <Tabs defaultValue="browse">
          <TabsList>
            <TabsTrigger value="browse">Browse</TabsTrigger>
            <TabsTrigger value="installed">
              Installed
              {installed && installed.length > 0 && (
                <Badge variant="secondary" className="ml-1.5 h-5 px-1.5 text-[10px]">
                  {installed.length}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="browse" className="space-y-4 mt-4">
            <div className="flex flex-wrap items-center gap-3">
              <div className="relative flex-1 min-w-[200px]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search items..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9"
                />
              </div>
              <div className="flex gap-1">
                {(["all", "skill", "command", "agent"] as const).map((t) => (
                  <Button
                    key={t}
                    variant={typeFilter === t ? "default" : "outline"}
                    size="sm"
                    onClick={() => setTypeFilter(t)}
                    className="capitalize"
                  >
                    {t === "all" ? "All" : `${t}s`}
                  </Button>
                ))}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setImportDialogOpen(true)}
              >
                <Plus className="w-4 h-4 mr-1" />
                Import Custom
              </Button>
            </div>

            {sourcesLoading ? (
              <div className="space-y-6">
                {Array.from({ length: 2 }).map((_, i) => (
                  <div key={i} className="space-y-3">
                    <Skeleton className="h-6 w-48" />
                    <div className="grid gap-2 grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
                      {Array.from({ length: 4 }).map((_, j) => (
                        <Skeleton key={j} className="h-16 w-full rounded-lg" />
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            ) : sources && sources.length > 0 ? (
              <div className="space-y-6">
                {sources.map((source, idx) => (
                  <SourceSection
                    key={source.url}
                    source={source}
                    sourceIndex={idx}
                    searchTerm={searchTerm}
                    typeFilter={typeFilter}
                    installedIds={installedIds}
                    onDirectInstall={handleDirectInstall}
                    onSelectItem={(item, src) => setSelectedItem({ item, source: src })}
                    installingId={installingId}
                  />
                ))}
              </div>
            ) : (
              <div className="text-center py-12 text-muted-foreground">
                <p className="text-sm">No marketplace sources configured.</p>
                <p className="text-xs mt-1">
                  Use &quot;Import Custom&quot; to import from any Git repository.
                </p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="installed" className="space-y-4 mt-4">
            <div className="flex flex-wrap items-center gap-3">
              <div className="relative flex-1 min-w-[200px]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search installed..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9"
                />
              </div>
              <div className="flex gap-1">
                {(["all", "skill", "command", "agent"] as const).map((t) => (
                  <Button
                    key={t}
                    variant={typeFilter === t ? "default" : "outline"}
                    size="sm"
                    onClick={() => setTypeFilter(t)}
                    className="capitalize"
                  >
                    {t === "all" ? "All" : `${t}s`}
                  </Button>
                ))}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setImportDialogOpen(true)}
              >
                <Plus className="w-4 h-4 mr-1" />
                Import Custom
              </Button>
            </div>

            {installedLoading ? (
              <div className="grid gap-2 grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full rounded-lg" />
                ))}
              </div>
            ) : filteredInstalled.length > 0 ? (
              <div className="grid gap-2 grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
                {filteredInstalled.map((item) => {
                  const isRemoving =
                    uninstallMutation.isPending &&
                    uninstallMutation.variables?.itemId === item.itemId;
                  return (
                    <div
                      key={item.itemId}
                      className="flex items-start gap-2 p-2.5 rounded-lg border"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-1.5">
                          <span className="text-sm font-medium truncate">
                            {item.itemName}
                          </span>
                          <Badge
                            variant="secondary"
                            className={`text-[10px] px-1.5 py-0 ${MARKETPLACE_CATEGORY_COLORS[item.itemType]}`}
                          >
                            {item.itemType}
                          </Badge>
                        </div>
                        <a
                          href={item.sourceUrl.replace(/\.git$/, "")}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-muted-foreground truncate mt-0.5 hover:text-blue-500 flex items-center gap-1"
                          onClick={(e) => e.stopPropagation()}
                        >
                          {item.sourceUrl.replace(/\.git$/, "").split("/").slice(-2).join("/")}
                          <ExternalLink className="w-2.5 h-2.5 shrink-0" />
                        </a>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 shrink-0"
                        onClick={() => handleUninstall(item.itemId, item.itemName)}
                        disabled={isRemoving}
                      >
                        {isRemoving ? (
                          <Loader2 className="w-3.5 h-3.5 animate-spin" />
                        ) : (
                          <Trash2 className="w-3.5 h-3.5" />
                        )}
                      </Button>
                    </div>
                  );
                })}
              </div>
            ) : installed && installed.length > 0 ? (
              <div className="text-center py-12 text-muted-foreground">
                <p className="text-sm">No items match your filters.</p>
              </div>
            ) : (
              <div className="text-center py-12 text-muted-foreground">
                <p className="text-sm">No items installed yet.</p>
                <p className="text-xs mt-1">
                  Browse the marketplace or import from a custom Git source.
                </p>
              </div>
            )}
          </TabsContent>
        </Tabs>
      </div>

      <ImportSourceDialog
        projectName={projectName}
        open={importDialogOpen}
        onOpenChange={setImportDialogOpen}
      />

      <Sheet open={!!selectedItem} onOpenChange={(open) => !open && setSelectedItem(null)}>
        <SheetContent className="sm:max-w-md">
          {selectedItem && (() => {
            const { item, source } = selectedItem;
            const isInstalled = installedIds.has(item.id);
            const isInstalling = installingId === item.id;
            const fileUrl = getSourceFileUrl(source, item.file_path);
            return (
              <>
                <SheetHeader>
                  <div className="flex items-center gap-2">
                    <SheetTitle className="text-lg">{item.name}</SheetTitle>
                    <Badge variant="secondary" className={MARKETPLACE_CATEGORY_COLORS[item.category]}>
                      {item.category}
                    </Badge>
                  </div>
                  <SheetDescription>{item.description}</SheetDescription>
                </SheetHeader>
                <div className="mt-6 space-y-4">
                  <div className="space-y-2">
                    <h4 className="text-sm font-medium">Source</h4>
                    <a
                      href={fileUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-sm text-blue-500 hover:underline flex items-center gap-1"
                    >
                      {item.file_path}
                      <ExternalLink className="w-3 h-3" />
                    </a>
                    <p className="text-xs text-muted-foreground">{source.name}</p>
                  </div>
                  {item.allowed_tools && item.allowed_tools.length > 0 && (
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium">Allowed Tools</h4>
                      <div className="flex flex-wrap gap-1">
                        {item.allowed_tools.map((tool) => (
                          <Badge key={tool} variant="outline" className="text-xs">
                            {tool}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                  <Button
                    className="w-full"
                    disabled={isInstalled || isInstalling}
                    onClick={() => {
                      handleDirectInstall(item, source);
                    }}
                  >
                    {isInstalling ? (
                      <>
                        <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                        Installing...
                      </>
                    ) : isInstalled ? (
                      <>
                        <Check className="w-4 h-4 mr-2" />
                        Installed
                      </>
                    ) : (
                      <>
                        <Plus className="w-4 h-4 mr-2" />
                        Install to Workspace
                      </>
                    )}
                  </Button>
                </div>
              </>
            );
          })()}
        </SheetContent>
      </Sheet>
    </div>
  );
}
