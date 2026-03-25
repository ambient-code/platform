"use client";

import { useState, useMemo } from "react";
import { useParams } from "next/navigation";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Download, Plus, Search, Trash2, Loader2 } from "lucide-react";
import { toast } from "sonner";
import {
  useMarketplaceSources,
  useMarketplaceCatalog,
  useInstalledItems,
  useUninstallItem,
} from "@/services/queries/use-marketplace";
import { ImportSourceDialog } from "@/components/import-source-dialog";
import type { MarketplaceCatalogItem, MarketplaceSource } from "@/types/marketplace";
import { MARKETPLACE_CATEGORY_COLORS } from "@/types/marketplace";

type TypeFilter = "all" | "skill" | "command" | "agent";

function SourceSection({
  source,
  sourceIndex,
  searchTerm,
  typeFilter,
  onImport,
}: {
  source: MarketplaceSource;
  sourceIndex: number;
  searchTerm: string;
  typeFilter: TypeFilter;
  onImport: (item: MarketplaceCatalogItem, source: MarketplaceSource) => void;
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
        <div className="grid gap-3 md:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 w-full rounded-lg" />
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
      <div className="grid gap-3 md:grid-cols-2">
        {filtered.map((item) => (
          <Card key={item.id} className="flex flex-col">
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <CardTitle className="text-sm">{item.name}</CardTitle>
                  <CardDescription className="text-xs mt-1 line-clamp-2">
                    {item.description}
                  </CardDescription>
                </div>
                <Badge variant="secondary" className={MARKETPLACE_CATEGORY_COLORS[item.category]}>
                  {item.category}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="pt-0 mt-auto">
              <Button
                variant="outline"
                size="sm"
                className="w-full"
                onClick={() => onImport(item, source)}
              >
                <Download className="w-3.5 h-3.5 mr-1.5" />
                Import
              </Button>
            </CardContent>
          </Card>
        ))}
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
  const [prefillUrl, setPrefillUrl] = useState("");
  const [prefillItems, setPrefillItems] = useState<string[]>([]);

  const { data: sources, isLoading: sourcesLoading } = useMarketplaceSources();
  const { data: installed, isLoading: installedLoading } = useInstalledItems(projectName);
  const uninstallMutation = useUninstallItem();

  const installedGrouped = useMemo(() => {
    if (!installed) return {};
    const groups: Record<string, typeof installed> = {};
    for (const item of installed) {
      const key = item.itemType;
      (groups[key] ??= []).push(item);
    }
    return groups;
  }, [installed]);

  const handleImportFromCatalog = (
    item: MarketplaceCatalogItem,
    source: MarketplaceSource
  ) => {
    setPrefillUrl(source.url);
    setPrefillItems([item.id]);
    setImportDialogOpen(true);
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
                onClick={() => {
                  setPrefillUrl("");
                  setPrefillItems([]);
                  setImportDialogOpen(true);
                }}
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
                    <div className="grid gap-3 md:grid-cols-2">
                      {Array.from({ length: 4 }).map((_, j) => (
                        <Skeleton key={j} className="h-24 w-full rounded-lg" />
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
                    onImport={handleImportFromCatalog}
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
            <div className="flex justify-end">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setPrefillUrl("");
                  setPrefillItems([]);
                  setImportDialogOpen(true);
                }}
              >
                <Plus className="w-4 h-4 mr-1" />
                Import Custom
              </Button>
            </div>

            {installedLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-16 w-full rounded-lg" />
                ))}
              </div>
            ) : installed && installed.length > 0 ? (
              <div className="space-y-4">
                {(["skill", "command", "agent", "workflow"] as const).map((type) => {
                  const items = installedGrouped[type];
                  if (!items || items.length === 0) return null;
                  return (
                    <div key={type} className="space-y-2">
                      <h3 className="text-sm font-medium uppercase tracking-wider text-muted-foreground capitalize">
                        {type}s ({items.length})
                      </h3>
                      {items.map((item) => {
                        const isRemoving =
                          uninstallMutation.isPending &&
                          uninstallMutation.variables?.itemId === item.itemId;
                        return (
                          <div
                            key={item.itemId}
                            className="flex items-center gap-3 p-3 rounded-lg border"
                          >
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                <span className="text-sm font-medium">
                                  {item.itemName}
                                </span>
                                <Badge
                                  variant="secondary"
                                  className={MARKETPLACE_CATEGORY_COLORS[item.itemType]}
                                >
                                  {item.itemType}
                                </Badge>
                              </div>
                              <p className="text-xs text-muted-foreground truncate mt-0.5">
                                {item.sourceUrl}
                              </p>
                            </div>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() =>
                                handleUninstall(item.itemId, item.itemName)
                              }
                              disabled={isRemoving}
                            >
                              {isRemoving ? (
                                <Loader2 className="w-4 h-4 animate-spin" />
                              ) : (
                                <Trash2 className="w-4 h-4" />
                              )}
                            </Button>
                          </div>
                        );
                      })}
                    </div>
                  );
                })}
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
        prefillUrl={prefillUrl}
        prefillItems={prefillItems}
        open={importDialogOpen}
        onOpenChange={setImportDialogOpen}
        onImported={() => {
          setPrefillUrl("");
          setPrefillItems([]);
        }}
      />
    </div>
  );
}
