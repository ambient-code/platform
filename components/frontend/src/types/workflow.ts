/**
 * Workflow-related types
 */

/**
 * Configuration for an out-of-the-box (OOTB) workflow
 * Represents a pre-defined workflow that can be activated in a session
 */
export type WorkflowConfig = {
  /** Unique identifier for the workflow */
  id: string;
  /** Display name shown in the UI */
  name: string;
  /** User-friendly description of what the workflow does */
  description: string;
  /** Git repository URL where the workflow code is stored */
  gitUrl: string;
  /** Git branch to use */
  branch: string;
  /** Optional path within the repository to the workflow files */
  path?: string;
  /** Whether this workflow is currently enabled and can be selected */
  enabled: boolean;
};

/**
 * A slash command available within a workflow
 */
export type WorkflowCommand = {
  /** Unique identifier for the command */
  id: string;
  /** Display name of the command */
  name: string;
  /** The actual slash command string (e.g., "/deploy") */
  slashCommand: string;
  /** Optional description of what the command does */
  description?: string;
  /** Optional icon identifier for the command */
  icon?: string;
};

/**
 * An agent available within a workflow
 */
export type WorkflowAgent = {
  /** Unique identifier for the agent */
  id: string;
  /** Display name of the agent */
  name: string;
  /** Optional description of the agent's capabilities */
  description?: string;
};

/**
 * Metadata about a workflow's capabilities
 * Includes available commands and agents
 */
export type WorkflowMetadata = {
  /** List of slash commands available in this workflow */
  commands: Array<WorkflowCommand>;
  /** List of agents available in this workflow */
  agents: Array<WorkflowAgent>;
};

/**
 * Selection criteria for activating a workflow
 * Used when creating or updating a session's active workflow
 */
export type WorkflowSelection = {
  /** Git repository URL where the workflow code is stored */
  gitUrl: string;
  /** Git branch to use */
  branch: string;
  /** Optional path within the repository to the workflow files */
  path?: string;
};
