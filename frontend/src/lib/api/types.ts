import { z } from "zod";

// Each API shape is declared once as a Zod schema; the TypeScript type is
// inferred from it so the two can never drift apart.

const EnvMapSchema = z.record(z.string(), z.string());

export const ProjectSchema = z.object({
  id: z.string(),
  name: z.string(),
});
export type Project = z.infer<typeof ProjectSchema>;

export const ProjectCreateRequestSchema = z.object({
  name: z.string(),
});
export type ProjectCreateRequest = z.infer<typeof ProjectCreateRequestSchema>;

export const ProjectUpdateRequestSchema = ProjectCreateRequestSchema.partial();
export type ProjectUpdateRequest = z.infer<typeof ProjectUpdateRequestSchema>;

export const ServiceSchema = z.object({
  id: z.string(),
  project_id: z.string(),
  name: z.string(),
  image: z.string(),
  port: z.number(),
  status: z.string(),
  public_domain: z.string(),
  private_domain: z.string(), // conceptually "internal_domain"
  env: EnvMapSchema,
  created_at: z.string(),
});
export type Service = z.infer<typeof ServiceSchema>;

export const ServiceCreateRequestSchema = z.object({
  name: z.string(),
  image: z.string(),
  port: z.number(),
  domain: z.string().nullable().optional(),
  env: z.string().optional(),
});
export type ServiceCreateRequest = z.infer<typeof ServiceCreateRequestSchema>;

export const ServiceUpdateRequestSchema = ServiceCreateRequestSchema.partial();
export type ServiceUpdateRequest = z.infer<typeof ServiceUpdateRequestSchema>;

export const DatabaseSchema = z.object({
  id: z.string(),
  project_id: z.string(),
  name: z.string(),
  engine: z.string(),
  port: z.number(),
  storage: z.number(),
  internal_domain: z.string(),
  env: EnvMapSchema,
  created_at: z.string(),
});
export type Database = z.infer<typeof DatabaseSchema>;

export const DatabaseCreateRequestSchema = z.object({
  name: z.string(),
  engine: z.string(),
  storage_gb: z.number(),
});
export type DatabaseCreateRequest = z.infer<typeof DatabaseCreateRequestSchema>;

export const DatabaseUpdateRequestSchema = DatabaseCreateRequestSchema.partial();
export type DatabaseUpdateRequest = z.infer<typeof DatabaseUpdateRequestSchema>;

export const VolumeSchema = z.object({
  id: z.string(),
  service_id: z.string(),
  mount_path: z.string(),
  storage_gb: z.number(),
  created_at: z.string(),
});
export type Volume = z.infer<typeof VolumeSchema>;

export const VolumeCreateRequestSchema = z.object({
  mount_path: z.string(),
  storage_gb: z.number(),
});
export type VolumeCreateRequest = z.infer<typeof VolumeCreateRequestSchema>;

export const ApiErrorSchema = z.object({
  error: z.string().optional(),
  message: z.string().optional(),
  issues: z
    .array(
      z.object({
        line: z.number().optional(),
        message: z.string(),
      }),
    )
    .optional(),
});
export type ApiError = z.infer<typeof ApiErrorSchema>;

export const BulkDeleteResponseSchema = z.object({
  deleted: z.array(z.string()),
  failed: z.array(
    z.object({
      id: z.string(),
      error: z.string(),
    }),
  ),
});
export type BulkDeleteResponse = z.infer<typeof BulkDeleteResponseSchema>;

export const ProjectConfigApplyRequestSchema = z.object({
  content: z.string(),
});
export type ProjectConfigApplyRequest = z.infer<typeof ProjectConfigApplyRequestSchema>;

export const GithubServiceSchema = z.object({
  id: z.string(),
  project_id: z.string(),
  name: z.string(),
  repo_name: z.string(),
  repo_id: z.number(),
  root_dir: z.string(),
  branch: z.string(),
  port: z.number(),
  status: z.string(),
  public_domain: z.string(),
  private_domain: z.string(),
  env: EnvMapSchema,
  created_at: z.string(),
});
export type GithubService = z.infer<typeof GithubServiceSchema>;

export const GithubServiceCreateRequestSchema = z.object({
  name: z.string(),
  repo: z.string(),
  repo_id: z.number(),
  port: z.number(),
  domain: z.string().nullable().optional(),
  root_dir: z.string(),
  branch: z.string().optional(),
  env: z.string().optional(),
});
export type GithubServiceCreateRequest = z.infer<typeof GithubServiceCreateRequestSchema>;

export const GithubServiceUpdateRequestSchema = GithubServiceCreateRequestSchema.pick({
  name: true,
  port: true,
  domain: true,
  env: true,
  branch: true,
}).partial();
export type GithubServiceUpdateRequest = z.infer<typeof GithubServiceUpdateRequestSchema>;

export const GithubConnectionSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  installation_id: z.number(),
  created_at: z.string(),
});
export type GithubConnection = z.infer<typeof GithubConnectionSchema>;

export const GithubConnectionCreateRequestSchema = z.object({
  installation_id: z.number(),
  // GitHub user-authorization code from the install redirect; the backend
  // exchanges it to prove the user can access the installation.
  code: z.string(),
});
export type GithubConnectionCreateRequest = z.infer<typeof GithubConnectionCreateRequestSchema>;

export const GithubRepoSchema = z.object({
  id: z.number(),
  name: z.string(),
  full_name: z.string(),
  clone_url: z.string(),
  html_url: z.string(),
  private: z.boolean(),
  default_branch: z.string(),
});
export type GithubRepo = z.infer<typeof GithubRepoSchema>;

export const GithubReposResponseSchema = z.object({
  total_count: z.number(),
  repositories: z.array(GithubRepoSchema),
});
export type GithubReposResponse = z.infer<typeof GithubReposResponseSchema>;

export const PreviewEnvironmentServiceSchema = z.object({
  id: z.string(),
  preview_env_id: z.string(),
  env_name: z.string(),
  source_type: z.string(),
  source_service_id: z.string(),
  github_service_id: z.string(),
  service_name: z.string(),
  repo_name: z.string(),
  engine: z.string(),
  pr: z.number(),
  pr_url: z.string(),
  branch: z.string(),
  hostname: z.string(),
  url: z.string(),
  env: EnvMapSchema,
  preview_url: z.string(),
  port: z.number(),
  status: z.string(),
  created_at: z.string(),
});
export type PreviewEnvironmentService = z.infer<typeof PreviewEnvironmentServiceSchema>;

export const PreviewEnvironmentSchema = z.object({
  id: z.string(),
  project_id: z.string(),
  name: z.string(),
  pr: z.number(),
  namespace: z.string(),
  created_at: z.string(),
});
export type PreviewEnvironment = z.infer<typeof PreviewEnvironmentSchema>;

export const PreviewEnvironmentServiceUpdateRequestSchema = z.object({
  env: z.string().optional(),
  port: z.number().optional(),
});
export type PreviewEnvironmentServiceUpdateRequest = z.infer<
  typeof PreviewEnvironmentServiceUpdateRequestSchema
>;

export const PreviewEnvironmentServicesSchema = PreviewEnvironmentSchema.extend({
  services: z.array(PreviewEnvironmentServiceSchema),
});
export type PreviewEnvironmentServices = z.infer<typeof PreviewEnvironmentServicesSchema>;
