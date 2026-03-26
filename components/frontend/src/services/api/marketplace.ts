import { apiClient } from "./client";
import type {
  MarketplaceSource,
  MarketplaceCatalogItem,
  InstalledItem,
  ScanResult,
} from "@/types/marketplace";

type ListSourcesResponse = {
  sources: MarketplaceSource[];
};

type CatalogResponse = {
  items: MarketplaceCatalogItem[];
};

type ListInstalledResponse = {
  items: InstalledItem[];
};

type ScanRequest = {
  url: string;
  branch: string;
  path?: string;
};

export async function listMarketplaceSources(): Promise<MarketplaceSource[]> {
  const response = await apiClient.get<ListSourcesResponse>("/marketplace/sources");
  return response.sources;
}

export async function getMarketplaceCatalog(
  sourceIndex: number
): Promise<MarketplaceCatalogItem[]> {
  const response = await apiClient.get<CatalogResponse>(
    `/marketplace/sources/${sourceIndex}/catalog`
  );
  return response.items ?? [];
}

export async function scanGitSource(projectName: string, request: ScanRequest): Promise<ScanResult> {
  return apiClient.post<ScanResult, ScanRequest>(`/projects/${projectName}/marketplace/scan`, request);
}

export async function listInstalledItems(
  projectName: string
): Promise<InstalledItem[]> {
  const response = await apiClient.get<ListInstalledResponse>(
    `/projects/${projectName}/marketplace/items`
  );
  return response.items;
}

export async function installItems(
  projectName: string,
  items: InstalledItem[]
): Promise<void> {
  await apiClient.post(`/projects/${projectName}/marketplace/items`, items);
}

export async function uninstallItem(
  projectName: string,
  itemId: string
): Promise<void> {
  await apiClient.delete(`/projects/${projectName}/marketplace/items/${itemId}`);
}
