export type AgenticSessionPhase = "Pending" | "Creating" | "Running" | "Stopping" | "Stopped" | "Completed" | "Failed";

export type LLMSettings = {
	model: string;
	temperature: number;
	maxTokens: number;
};

// Generic repo type used by RFE workflows (retains optional clonePath)
export type GitRepository = {
    url: string;
    branch?: string;
};

// Simplified multi-repo session mapping
export type SessionRepo = {
    url: string;
    branch?: string;
    autoPush?: boolean;
};

export type AgenticSessionSpec = {
	initialPrompt?: string;
	llmSettings: LLMSettings;
	timeout: number;
	displayName?: string;
	project?: string;
	interactive?: boolean;
	// Multi-repo support
	repos?: SessionRepo[];
	// Active workflow for dynamic workflow switching
	activeWorkflow?: {
		gitUrl: string;
		branch: string;
		path?: string;
	};
};

export type ReconciledRepo = {
	url: string;
	branch: string; // DEPRECATED: Use currentActiveBranch instead
	name?: string;
	branches?: string[]; // All local branches available
	currentActiveBranch?: string; // Currently checked out branch
	defaultBranch?: string; // Default branch of remote
	status?: "Cloning" | "Ready" | "Failed";
	clonedAt?: string;
};

export type ReconciledWorkflow = {
	gitUrl: string;
	branch: string;
	path?: string;
	status?: "Cloning" | "Active" | "Failed";
	appliedAt?: string;
};

export type SessionCondition = {
	type: string;
	status: "True" | "False" | "Unknown";
	reason?: string;
	message?: string;
	lastTransitionTime?: string;
	observedGeneration?: number;
};

export type AgenticSessionStatus = {
	observedGeneration?: number;
	phase: AgenticSessionPhase;
	startTime?: string;
	completionTime?: string;
	reconciledRepos?: ReconciledRepo[];
	reconciledWorkflow?: ReconciledWorkflow;
	sdkSessionId?: string;
	sdkRestartCount?: number;
	conditions?: SessionCondition[];
};

export type AgenticSession = {
	metadata: {
		name: string;
		namespace: string;
		creationTimestamp: string;
		uid: string;
		labels?: Record<string, string>;
		annotations?: Record<string, string>;
	};
	spec: AgenticSessionSpec;
	status?: AgenticSessionStatus;
};

export type CreateAgenticSessionRequest = {
	initialPrompt?: string;
	llmSettings?: Partial<LLMSettings>;
	displayName?: string;
	timeout?: number;
	project?: string;
	parent_session_id?: string;
  	environmentVariables?: Record<string, string>;
	interactive?: boolean;
	// Multi-repo support
	repos?: SessionRepo[];
	labels?: Record<string, string>;
	annotations?: Record<string, string>;
};

export type AgentPersona = {
	persona: string;
	name: string;
	role: string;
	description: string;
};

export type { Project } from "@/types/project";
