import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as marketplaceApi from "@/services/api/marketplace";
import type { InstalledItem, ScanResult } from "@/types/marketplace";

export const marketplaceKeys = {
  all: ["marketplace"] as const,
  sources: () => [...marketplaceKeys.all, "sources"] as const,
  catalog: (sourceIndex: number) =>
    [...marketplaceKeys.all, "catalog", sourceIndex] as const,
  installed: (projectName: string) =>
    [...marketplaceKeys.all, "installed", projectName] as const,
};

export function useMarketplaceSources() {
  return useQuery({
    queryKey: marketplaceKeys.sources(),
    queryFn: marketplaceApi.listMarketplaceSources,
    staleTime: 5 * 60 * 1000,
  });
}

export function useMarketplaceCatalog(sourceIndex: number) {
  return useQuery({
    queryKey: marketplaceKeys.catalog(sourceIndex),
    queryFn: () => marketplaceApi.getMarketplaceCatalog(sourceIndex),
    enabled: sourceIndex >= 0,
    staleTime: 5 * 60 * 1000,
  });
}

export function useInstalledItems(projectName: string) {
  return useQuery({
    queryKey: marketplaceKeys.installed(projectName),
    queryFn: () => marketplaceApi.listInstalledItems(projectName),
    enabled: !!projectName,
    staleTime: 60 * 1000,
  });
}

export function useInstallItems() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectName,
      items,
    }: {
      projectName: string;
      items: InstalledItem[];
    }) => marketplaceApi.installItems(projectName, items),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: marketplaceKeys.installed(variables.projectName),
      });
    },
  });
}

export function useUninstallItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectName,
      itemId,
    }: {
      projectName: string;
      itemId: string;
    }) => marketplaceApi.uninstallItem(projectName, itemId),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({
        queryKey: marketplaceKeys.installed(variables.projectName),
      });
    },
  });
}

export function useScanGitSource() {
  return useMutation<
    ScanResult,
    Error,
    { projectName: string; url: string; branch: string; path?: string }
  >({
    mutationFn: ({ projectName, ...request }) => marketplaceApi.scanGitSource(projectName, request),
  });
}
