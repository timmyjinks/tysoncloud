// Mirrors backend/server/model.go response/request shapes.

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
  created_at: string;
};

export type ServiceCreateRequest = {
  name: string;
  image: string;
  port: number;
};

export type ServiceUpdateRequest = {
  name?: string;
  image?: string;
  port?: number;
};

export type Database = {
  id: string;
  project_id: string;
  name: string;
  engine: string;
  port: number;
  storage: number;
  internal_domain: string;
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
};
