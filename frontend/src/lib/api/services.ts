import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";
import type { BulkDeleteResponse, Service, ServiceCreateRequest, ServiceUpdateRequest } from "./types";

export const serviceKeys = {
  byProject: (projectId: string) => ["projects", projectId, "services"] as const,
  detail: (id: string) => ["services", id] as const,
};

export function useServices(projectId: string) {
  return useQuery({
    queryKey: serviceKeys.byProject(projectId),
    queryFn: () => api.get<Service[]>(`/projects/${projectId}/services`),
    enabled: !!projectId,
  });
}

export function useService(id: string, opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: serviceKeys.detail(id),
    queryFn: () => api.get<Service>(`/services/${id}`),
    enabled: !!id && (opts?.enabled ?? true),
  });
}

export function useCreateService(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: ServiceCreateRequest) =>
      api.post<void>(`/projects/${projectId}/services`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: serviceKeys.byProject(projectId) }),
  });
}

export function useUpdateService(projectId: string, serviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: ServiceUpdateRequest) =>
      api.put<void>(`/projects/${projectId}/services/${serviceId}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: serviceKeys.byProject(projectId) });
      qc.invalidateQueries({ queryKey: serviceKeys.detail(serviceId) });
    },
  });
}

export function useDeleteService(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (serviceId: string) =>
      api.delete<void>(`/projects/${projectId}/services/${serviceId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: serviceKeys.byProject(projectId) }),
  });
}

export function useDeleteServices(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (serviceIds: string[]) =>
      api.delete<BulkDeleteResponse>(`/projects/${projectId}/services`, { ids: serviceIds }),
    onSuccess: () => qc.invalidateQueries({ queryKey: serviceKeys.byProject(projectId) }),
  });
}
