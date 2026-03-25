export type MarketplaceSource = {
  name: string;
  url: string;
  branch: string;
  path?: string;
  catalogUrl: string;
  description?: string;
};

export type MarketplaceCatalogItem = {
  id: string;
  name: string;
  description: string;
  category: "skill" | "command" | "agent";
  file_path: string;
  allowed_tools?: string[];
};

export type InstalledItem = {
  sourceUrl: string;
  sourceBranch: string;
  sourcePath?: string;
  itemId: string;
  itemType: "skill" | "command" | "agent" | "workflow";
  itemName: string;
  filePath: string;
};

export type ScanResult = {
  items: DiscoveredItem[];
  isWorkflow: boolean;
  hasClaudeMd: boolean;
  workflowName?: string;
  workflowDescription?: string;
};

export type DiscoveredItem = {
  id: string;
  name: string;
  description: string;
  type: "skill" | "command" | "agent";
  filePath: string;
};

export const MARKETPLACE_CATEGORY_COLORS: Record<string, string> = {
  skill: "bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200",
  command: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  agent: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  workflow: "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200",
};
