import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as projectSettingsApi from "../api/project-settings";
import type { UpdateProjectSettingsRequest } from "@/types/project-settings";

export function useProjectSettings(projectName: string) {
  return useQuery({
    queryKey: ["project-settings", projectName],
    queryFn: () => projectSettingsApi.getProjectSettings(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateProjectSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: UpdateProjectSettingsRequest;
    }) => projectSettingsApi.updateProjectSettings(projectName, data),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({
        queryKey: ["project-settings", projectName],
      });
    },
  });
}
