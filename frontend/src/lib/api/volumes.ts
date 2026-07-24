import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, ApiRequestError } from "./client";
import type { Volume, VolumeCreateRequest } from "./types";

export const volumeKeys = {
  byService: (serviceId: string) => ["services", serviceId, "volume"] as const,
};

export function useVolume(serviceId: string) {
  return useQuery({
    queryKey: volumeKeys.byService(serviceId),
    queryFn: async () => {
      try {
        return await api.get<Volume>(`/services/${serviceId}/volumes`);
      } catch (err) {
        // No volume attached yet — treat 404 as "none", not an error state.
        if (err instanceof ApiRequestError && err.status === 404) return null;
        throw err;
      }
    },
    enabled: !!serviceId,
  });
}

export function useAttachVolume(projectId: string, serviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: VolumeCreateRequest) =>
      api.post<void>(`/projects/${projectId}/services/${serviceId}/volumes`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: volumeKeys.byService(serviceId) }),
  });
}

export function useDetachVolume(projectId: string, serviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      api.delete<void>(`/projects/${projectId}/services/${serviceId}/volumes`),
    onSuccess: () => qc.invalidateQueries({ queryKey: volumeKeys.byService(serviceId) }),
  });
}
