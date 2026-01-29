"use client";

import { useCallback } from "react";
import { 
  useGitStatus,
  useConfigureGitRemote,
} from "@/services/queries/use-workspace";
import { successToast, errorToast } from "@/hooks/use-toast";

type UseGitOperationsProps = {
  projectName: string;
  sessionName: string;
  directoryPath: string;
  remoteBranch?: string;
};

export function useGitOperations({
  projectName,
  sessionName,
  directoryPath,
}: UseGitOperationsProps) {
  const configureRemoteMutation = useConfigureGitRemote();
  
  // Use React Query for git status
  const { data: gitStatus, refetch: fetchGitStatus } = useGitStatus(
    projectName,
    sessionName,
    directoryPath,
    { enabled: !!projectName && !!sessionName && !!directoryPath }
  );

  // Configure remote for the directory
  const configureRemote = useCallback(async (remoteUrl: string, branch: string) => {
    try {
      await configureRemoteMutation.mutateAsync({
        projectName,
        sessionName,
        path: directoryPath,
        remoteUrl: remoteUrl.trim(),
        branch: branch.trim(),
      });
      
      successToast("Remote configured successfully");
      await fetchGitStatus();
      
      return true;
    } catch (error) {
      console.error("Failed to configure remote:", error);
      errorToast(error instanceof Error ? error.message : "Failed to configure remote");
      return false;
    }
  }, [projectName, sessionName, directoryPath, configureRemoteMutation, fetchGitStatus]);

  // Removed: handleGitPull, handleGitPush, handleGitSynchronize, handleCommit
  // Agent handles all git operations now

  return {
    gitStatus,
    fetchGitStatus,
    configureRemote,
    isConfiguringRemote: configureRemoteMutation.isPending,
  };
}

