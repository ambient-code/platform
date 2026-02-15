import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as httpToolsApi from '../api/http-tools';

export function useHttpTools(projectName: string) {
  return useQuery({
    queryKey: ['http-tools', projectName],
    queryFn: () => httpToolsApi.getHttpTools(projectName),
    enabled: !!projectName,
  });
}

export function useUpdateHttpTools() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectName,
      data,
    }: {
      projectName: string;
      data: httpToolsApi.HttpToolsData;
    }) => httpToolsApi.updateHttpTools(projectName, data),
    onSuccess: (_, { projectName }) => {
      queryClient.invalidateQueries({ queryKey: ['http-tools', projectName] });
    },
  });
}
