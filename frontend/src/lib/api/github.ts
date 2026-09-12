import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";
import type {
  BulkDeleteResponse,
  GithubConnection,
  GithubConnectionCreateRequest,
  GithubReposResponse,
  GithubService,
  GithubServiceCreateRequest,
  GithubServiceUpdateRequest,
} from "./types";

export const githubKeys = {
  connections: ["github", "connections"] as const,
  app: ["github", "app"] as const,
  repos: (installationId: string) => ["github", "installation", installationId, "repositories"] as const,
  byProject: (projectId: string) => ["projects", projectId, "github_services"] as const,
  detail: (id: string) => ["github_services", id] as const,
};

export type GithubAppInfo = {
  slug: string;
  install_url: string;
};

export function useGithubApp() {
  return useQuery({
    queryKey: githubKeys.app,
    queryFn: () => api.get<GithubAppInfo>("/github/app"),
  });
}

export function useGithubConnections() {
  return useQuery({
    queryKey: githubKeys.connections,
    queryFn: () => api.get<GithubConnection[]>("/github/connections"),
  });
}

export function useCreateGithubConnection() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: GithubConnectionCreateRequest) =>
      api.post<GithubConnection>("/github/connections", body),
    onSuccess: () => qc.invalidateQueries({ queryKey: githubKeys.connections }),
  });
}

export function useDeleteGithubConnection() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (connectionId: string) => api.delete<void>(`/github/connections/${connectionId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: githubKeys.connections }),
  });
}

export function useGithubRepos(installationId: string) {
  return useQuery({
    queryKey: githubKeys.repos(installationId),
    queryFn: () => api.get<GithubReposResponse>(`/github/installations/${installationId}/repositories`),
    enabled: !!installationId,
  });
}

export function useGithubServices(projectId: string) {
  return useQuery({
    queryKey: githubKeys.byProject(projectId),
    queryFn: () => api.get<GithubService[]>(`/projects/${projectId}/github_services`),
    enabled: !!projectId,
  });
}

export function useGithubService(id: string) {
  return useQuery({
    queryKey: githubKeys.detail(id),
    queryFn: () => api.get<GithubService>(`/github_services/${id}`),
    enabled: !!id,
  });
}

export function useCreateGithubService(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: GithubServiceCreateRequest) =>
      api.post<GithubService>(`/projects/${projectId}/github_services`, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: githubKeys.byProject(projectId) }),
  });
}

export function useUpdateGithubService(projectId: string, githubServiceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: GithubServiceUpdateRequest) =>
      api.put<GithubService>(`/projects/${projectId}/github_services/${githubServiceId}`, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: githubKeys.byProject(projectId) });
      qc.invalidateQueries({ queryKey: githubKeys.detail(githubServiceId) });
    },
  });
}

export function useDeleteGithubService(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (githubServiceId: string) =>
      api.delete<void>(`/projects/${projectId}/github_services/${githubServiceId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: githubKeys.byProject(projectId) }),
  });
}

export function useDeleteGithubServices(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ids: string[]) =>
      api.delete<BulkDeleteResponse>(`/projects/${projectId}/github_services`, { ids }),
    onSuccess: () => qc.invalidateQueries({ queryKey: githubKeys.byProject(projectId) }),
  });
}
