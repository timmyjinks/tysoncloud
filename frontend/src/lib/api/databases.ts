import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";
import type { Database, DatabaseCreateRequest, DatabaseUpdateRequest } from "./types";

export const databaseKeys = {
  byProject: (projectId: string) => ["projects", projectId, "databases"] as const,
  detail: (id: string) => ["databases", id] as const,
};

export function useDatabases(projectId: string) {
  return useQuery({
    queryKey: databaseKeys.byProject(projectId),
    queryFn: () => api.get<Database[]>(`/projects/${projectId}/databases`),
    enabled: !!projectId,
  });
}

export function useDatabase(id: string) {
  return useQuery({
    queryKey: databaseKeys.detail(id),
    queryFn: () => api.get<Database>(`/databases/${id}`),
    enabled: !!id,
  });
}

export function useCreateDatabase(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: DatabaseCreateRequest) =>
      api.post<void>(`/projects/${projectId}/databases`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: databaseKeys.byProject(projectId) }),
  });
}

export function useUpdateDatabase(projectId: string, databaseId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: DatabaseUpdateRequest) =>
      api.put<void>(`/projects/${projectId}/databases/${databaseId}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: databaseKeys.byProject(projectId) });
      qc.invalidateQueries({ queryKey: databaseKeys.detail(databaseId) });
    },
  });
}

export function useDeleteDatabase(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (databaseId: string) =>
      api.delete<void>(`/projects/${projectId}/databases/${databaseId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: databaseKeys.byProject(projectId) }),
  });
}
