// Mirrors the per-container limits in backend/kubernetes/deployment.go.
export const SERVICE_RESOURCE_LIMITS = {
  cpu: "0.5 vCPU",
  memory: "1 GiB",
} as const;

export const SERVICE_RESOURCE_SUMMARY = `${SERVICE_RESOURCE_LIMITS.cpu} · ${SERVICE_RESOURCE_LIMITS.memory} per instance`;
