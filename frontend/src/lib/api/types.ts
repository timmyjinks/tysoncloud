export type Project = {
  id: string;
  name: string;
};

export type ProjectCreateRequest = {
  name: string;
};

export type ProjectUpdateRequest = {
  name?: string;
};

export type Service = {
  id: string;
  project_id: string;
  name: string;
  image: string;
  port: number;
  status: string;
  public_domain: string;
  private_domain: string; // conceptually "internal_domain"
  env: Record<string, string>;
  created_at: string;
};

export type ServiceCreateRequest = {
  name: string;
  image: string;
  port: number;
  domain?: string | null;
  env?: string;
};

export type ServiceUpdateRequest = {
  name?: string;
  image?: string;
  port?: number;
  domain?: string | null;
  env?: string;
};

export type Database = {
  id: string;
  project_id: string;
  name: string;
  engine: string;
  port: number;
  storage: number;
  internal_domain: string;
  env: Record<string, string>;
  created_at: string;
};

export type DatabaseCreateRequest = {
  name: string;
  engine: string;
  storage_gb: number;
};

export type DatabaseUpdateRequest = {
  name?: string;
  engine?: string;
  storage_gb?: number;
};

export type Volume = {
  id: string;
  service_id: string;
  mount_path: string;
  storage_gb: number;
  created_at: string;
};

export type VolumeCreateRequest = {
  mount_path: string;
  storage_gb: number;
};

export type ApiError = {
  error?: string;
  message?: string;
  issues?: { line?: number; message: string }[];
};

export type BulkDeleteResponse = {
  deleted: string[];
  failed: { id: string; error: string }[];
};

export type ProjectConfigApplyRequest = {
  content: string;
};

export type GithubService = {
  id: string;
  project_id: string;
  name: string;
  repo: string;
  repo_id: string;
  root_dir: string;
  port: number;
  status: string;
  public_domain: string;
  private_domain: string;
  env: Record<string, string>;
  created_at: string;
};

export type GithubServiceCreateRequest = {
  name: string;
  repo: string;
  repo_id: string;
  port: number;
  domain?: string | null;
  root_dir: string;
  env?: string;
};

export type GithubServiceUpdateRequest = {
  name?: string;
  port?: number;
  domain?: string | null;
  env?: string;
};

export type GithubConnection = {
  id: string;
  user_id: string;
  installation_id: string;
  created_at: string;
};

export type GithubConnectionCreateRequest = {
  installation_id: string;
};

export type GithubRepo = {
  id: number;
  name: string;
  full_name: string;
  clone_url: string;
  html_url: string;
  private: boolean;
  default_branch: string;
};

export type GithubReposResponse = {
  total_count: number;
  repositories: GithubRepo[];
};
