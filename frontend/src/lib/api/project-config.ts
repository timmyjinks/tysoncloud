import { useMutation } from "@tanstack/react-query";
import { api } from "./client";
import type { ProjectConfigApplyRequest } from "./types";

export function useApplyProjectConfig(projectId: string) {
  return useMutation({
    mutationFn: (body: ProjectConfigApplyRequest) =>
      api.post<void>(`/projects/${projectId}/config`, body),
  });
}
