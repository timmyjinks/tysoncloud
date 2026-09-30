import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";
import type {
  PreviewEnvironment,
  PreviewEnvListItem,
  PreviewEnvSummary,
  PreviewServiceUpdateRequest,
} from "./types";

export const previewKeys = {
  byProject: (projectId: string) => ["projects", projectId, "preview_environments"] as const,
  detail: (projectId: string, envId: string) =>
    ["projects", projectId, "preview_environments", envId] as const,
};

export function useProjectPreviewEnvironments(projectId: string) {
  return useQuery({
    queryKey: previewKeys.byProject(projectId),
    queryFn: () => api.get<PreviewEnvSummary[]>(`/projects/${projectId}/preview_environments`),
    enabled: !!projectId,
  });
}

export function useProjectPreviewEnvironment(projectId: string, envId: string | null) {
  return useQuery({
    queryKey: previewKeys.detail(projectId, envId ?? ""),
    queryFn: () =>
      api.get<PreviewEnvListItem>(`/projects/${projectId}/preview_environments/${envId}`),
    enabled: !!projectId && !!envId,
  });
}

export function useUpdatePreviewService(projectId: string, envId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ sourceId, body }: { sourceId: string; body: PreviewServiceUpdateRequest }) =>
      api.put<PreviewEnvironment>(
        `/projects/${projectId}/preview_environments/${envId}/services/${sourceId}`,
        body,
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: previewKeys.detail(projectId, envId) });
    },
  });
}

export function useDeletePreviewService(projectId: string, envId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (sourceId: string) =>
      api.delete<void>(
        `/projects/${projectId}/preview_environments/${envId}/services/${sourceId}`,
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: previewKeys.detail(projectId, envId) });
      qc.invalidateQueries({ queryKey: previewKeys.byProject(projectId) });
    },
  });
}
