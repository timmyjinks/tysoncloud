import { useEffect, useMemo, useRef, useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import {
  ChevronDown,
  Database as DatabaseIcon,
  FileCode,
  Github,
  Pencil,
  Plus,
  Server,
  Trash2,
} from "lucide-react";
import { useProject } from "@/lib/api/projects";
import { useDeleteService, useDeleteServices, useServices } from "@/lib/api/services";
import { useDatabases, useDeleteDatabase, useDeleteDatabases } from "@/lib/api/databases";
import {
  useDeleteGithubService,
  useDeleteGithubServices,
  useGithubServices,
} from "@/lib/api/github";
import {
  useProjectPreviewEnvironment,
  useProjectPreviewEnvironments,
  useUpdatePreviewService,
} from "@/lib/api/previews";
import { ApiRequestError, getErrorMessage } from "@/lib/api/client";
import { formatEnvLines } from "@/lib/utils";
import { ResourceRow } from "@/components/resource-row";
import { ResourceStatusBar } from "@/components/resource-status-bar";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";
import { BulkDeleteConfirmDialog } from "@/components/bulk-delete-confirm-dialog";
import { ErrorBanner } from "@/components/error-banner";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { SERVICE_RESOURCE_LIMITS } from "@/lib/resource-limits";
import { Checkbox } from "@/components/ui/checkbox";
import type { PreviewEnvironment, Service, Database, GithubService } from "@/lib/api/types";

export const Route = createFileRoute("/projects/$projectId/")({
  component: ProjectDetail,
});

type Resource =
  | { kind: "service"; data: Service }
  | { kind: "database"; data: Database }
  | { kind: "github_service"; data: GithubService };

function ProjectDetail() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate();
  const { data: project } = useProject(projectId);
  const {
    data: services,
    isLoading: servicesLoading,
    error: servicesError,
    refetch: refetchServices,
  } = useServices(projectId);
  const {
    data: databases,
    isLoading: databasesLoading,
    error: databasesError,
    refetch: refetchDatabases,
  } = useDatabases(projectId);
  const {
    data: githubServices,
    isLoading: githubServicesLoading,
    error: githubServicesError,
    refetch: refetchGithubServices,
  } = useGithubServices(projectId);

  const deleteService = useDeleteService(projectId);
  const deleteDatabase = useDeleteDatabase(projectId);
  const deleteServices = useDeleteServices(projectId);
  const deleteDatabases = useDeleteDatabases(projectId);
  const deleteGithubService = useDeleteGithubService(projectId);
  const deleteGithubServices = useDeleteGithubServices(projectId);
  const [pendingService, setPendingService] = useState<Service | null>(null);
  const [pendingDatabase, setPendingDatabase] = useState<Database | null>(null);
  const [pendingGithubService, setPendingGithubService] = useState<GithubService | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [pendingBulk, setPendingBulk] = useState<Resource[] | null>(null);
  const [bulkError, setBulkError] = useState<string | null>(null);
  // Environment scope: production is the live resource list below; any other
  // value is a preview env id whose service copies replace it.
  const [selectedEnv, setSelectedEnv] = useState("production");
  const [editingPreview, setEditingPreview] = useState<PreviewEnvironment | null>(null);
  const {
    data: previewEnvs,
    error: previewEnvsError,
    refetch: refetchPreviewEnvs,
  } = useProjectPreviewEnvironments(projectId);
  const previewDetail = useProjectPreviewEnvironment(
    projectId,
    selectedEnv === "production" ? null : selectedEnv,
  );
  useEffect(() => {
    if (
      selectedEnv !== "production" &&
      previewDetail.error instanceof ApiRequestError &&
      previewDetail.error.status === 404
    ) {
      setSelectedEnv("production");
    }
  }, [previewDetail.error, selectedEnv]);
  const selectedPreviewEnv = previewDetail.data ?? null;
  const isPreviewMode = selectedEnv !== "production";

  const isLoading = servicesLoading || databasesLoading || githubServicesLoading;

  const resources = useMemo<Resource[]>(() => {
    const items: Resource[] = [
      ...(services ?? []).map((s) => ({ kind: "service" as const, data: s })),
      ...(databases ?? []).map((d) => ({ kind: "database" as const, data: d })),
      ...(githubServices ?? []).map((g) => ({ kind: "github_service" as const, data: g })),
    ];
    return items.sort(
      (a, b) => new Date(b.data.created_at).getTime() - new Date(a.data.created_at).getTime(),
    );
  }, [services, databases, githubServices]);

  const runningCount =
    (services ?? []).filter((s) => s.status === "running").length +
    (githubServices ?? []).filter((s) => s.status === "running").length;

  const allSelected = resources.length > 0 && selectedIds.size === resources.length;

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const toggleSelectAll = () => {
    setSelectedIds(allSelected ? new Set() : new Set(resources.map((r) => r.data.id)));
  };

  const selectedResources = resources.filter((r) => selectedIds.has(r.data.id));
  const bulkPending =
    deleteServices.isPending || deleteDatabases.isPending || deleteGithubServices.isPending;

  const onBulkConfirm = () => {
    if (!pendingBulk) return;
    const remaining = new Set(selectedIds);
    const serviceIds = pendingBulk
      .filter((r): r is Resource & { kind: "service" } => r.kind === "service")
      .map((r) => r.data.id);
    const databaseIds = pendingBulk
      .filter((r): r is Resource & { kind: "database" } => r.kind === "database")
      .map((r) => r.data.id);
    const githubServiceIds = pendingBulk
      .filter((r): r is Resource & { kind: "github_service" } => r.kind === "github_service")
      .map((r) => r.data.id);

    if (serviceIds.length > 0) {
      deleteServices.mutate(serviceIds, {
        onSuccess: (data) => {
          data.deleted.forEach((id) => remaining.delete(id));
          setSelectedIds(new Set(remaining));
          if (data.failed.length > 0) {
            setBulkError(
              `Couldn't delete ${data.failed.length} service${data.failed.length === 1 ? "" : "s"}.`,
            );
          } else if (remaining.size === 0 && githubServiceIds.length === 0 && databaseIds.length === 0) {
            setPendingBulk(null);
            setBulkError(null);
          } else if (remaining.size === 0) {
            // wait for other mutations
          }
        },
      });
    }

    if (databaseIds.length > 0) {
      deleteDatabases.mutate(databaseIds, {
        onSuccess: (data) => {
          data.deleted.forEach((id) => remaining.delete(id));
          setSelectedIds(new Set(remaining));
          if (data.failed.length > 0) {
            setBulkError(
              `Couldn't delete ${data.failed.length} database${data.failed.length === 1 ? "" : "s"}.`,
            );
          } else if (remaining.size === 0) {
            setPendingBulk(null);
            setBulkError(null);
          }
        },
      });
    }

    if (githubServiceIds.length > 0) {
      deleteGithubServices.mutate(githubServiceIds, {
        onSuccess: (data) => {
          data.deleted.forEach((id) => remaining.delete(id));
          setSelectedIds(new Set(remaining));
          if (data.failed.length > 0) {
            setBulkError(
              `Couldn't delete ${data.failed.length} GitHub service${data.failed.length === 1 ? "" : "s"}.`,
            );
          } else if (remaining.size === 0) {
            setPendingBulk(null);
            setBulkError(null);
          }
        },
      });
    }
  };

  return (
    <main className="mx-auto max-w-6xl px-4 py-10 sm:px-6 lg:px-8">
      <PageHeader
        title={
          <span className="inline-flex items-center gap-3">
            {project?.name ?? projectId}
            <Link
              to="/projects/$projectId/edit"
              params={{ projectId }}
              aria-label="Rename project"
              className="text-[var(--color-text-faint)] transition-colors hover:text-[var(--color-accent)]"
            >
              <Pencil className="h-5 w-5" />
            </Link>
          </span>
        }
        description="Everything deployed in this project"
      >
        <Link to="/projects/$projectId/config" params={{ projectId }}>
          <Button variant="outline" size="sm">
            <FileCode className="h-4 w-4" />
            Config
          </Button>
        </Link>
      </PageHeader>

      <div className="mt-10 mb-4 flex flex-wrap items-end justify-between gap-4">
        <div className="flex items-center gap-3">
          {!isPreviewMode && resources.length > 0 && (
            <Checkbox
              aria-label={allSelected ? "Deselect all resources" : "Select all resources"}
              checked={allSelected}
              onChange={toggleSelectAll}
            />
          )}
          <h2 className="text-lg font-medium text-[var(--color-text-muted)]">
            {isPreviewMode ? (
              <>
                {selectedPreviewEnv?.name ?? "Preview"}{" "}
                {selectedPreviewEnv && (
                  <span className="text-[var(--color-text-faint)]">
                    · {selectedPreviewEnv.services.length} preview service
                    {selectedPreviewEnv.services.length === 1 ? "" : "s"}
                  </span>
                )}
              </>
            ) : (
              <>
                Resources{" "}
                <span className="text-[var(--color-text-faint)]">· {resources.length} total</span>
              </>
            )}
          </h2>
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <div className="w-64">
            <Label htmlFor="environment">Environment</Label>
            <Select
              id="environment"
              value={selectedEnv}
              onChange={(e) => setSelectedEnv(e.target.value)}
              className="mt-2"
            >
              <option value="production">Production</option>
              {(previewEnvs ?? []).map((e) => (
                <option key={e.id} value={e.id}>
                  {e.name}
                </option>
              ))}
            </Select>
          </div>
          {!isPreviewMode && (
            <DropdownMenu
              trigger={
                <Button size="sm">
                  <Plus className="h-4 w-4" />
                  New
                  <ChevronDown className="h-4 w-4 opacity-60" />
                </Button>
              }
            >
              <DropdownMenuItem
                onClick={() =>
                  navigate({
                    to: "/projects/$projectId/services/new",
                    params: { projectId },
                  })
                }
              >
                <Server className="h-4 w-4" />
                Service
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() =>
                  navigate({
                    to: "/projects/$projectId/github_services/new",
                    params: { projectId },
                  })
                }
              >
                <Github className="h-4 w-4" />
                GitHub Service
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() =>
                  navigate({
                    to: "/projects/$projectId/databases/new",
                    params: { projectId },
                  })
                }
              >
                <DatabaseIcon className="h-4 w-4" />
                Database
              </DropdownMenuItem>
            </DropdownMenu>
          )}
        </div>
      </div>

      {isPreviewMode && selectedPreviewEnv && (
        <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md bg-[var(--color-surface-2)] px-5 py-3 text-sm text-[var(--color-text-muted)]">
          <span>
            Ephemeral preview — managed by the pull request. Env and port edits apply
            here only and reset on the next sync.
          </span>
          {selectedPreviewEnv.services[0]?.pr_url && (
            <a
              href={selectedPreviewEnv.services[0].pr_url}
              target="_blank"
              rel="noreferrer"
              className="font-mono text-[var(--color-accent)] hover:text-[var(--color-accent-hover)]"
            >
              #{selectedPreviewEnv.pr} ↗
            </a>
          )}
          <span className="font-mono text-[var(--color-text-faint)]">
            {selectedPreviewEnv.namespace}
          </span>
        </div>
      )}

      {previewEnvsError && (
        <ErrorBanner
          className="mb-4"
          message={getErrorMessage(previewEnvsError)}
          onRetry={() => refetchPreviewEnvs()}
          retryLabel="Retry"
        />
      )}

      {!isPreviewMode && selectedIds.size > 0 && (
        <div className="mb-4 flex items-center justify-between gap-4 rounded-md bg-[var(--color-surface-2)] px-5 py-3">
          <span className="font-mono text-sm text-[var(--color-text-muted)]">
            {selectedIds.size} selected
          </span>
          <Button
            variant="danger"
            size="sm"
            onClick={() => {
              setPendingBulk(selectedResources);
              setBulkError(null);
            }}
          >
            <Trash2 className="h-4 w-4" />
            Delete selected
          </Button>
        </div>
      )}

      {!isPreviewMode && isLoading && (
        <p className="text-base text-[var(--color-text-faint)]">loading resources…</p>
      )}

      {!isPreviewMode && servicesError && (
        <ErrorBanner
          className="mb-4"
          message={getErrorMessage(servicesError)}
          onRetry={() => refetchServices()}
          retryLabel="Retry"
        />
      )}

      {!isPreviewMode && databasesError && (
        <ErrorBanner
          className="mb-4"
          message={getErrorMessage(databasesError)}
          onRetry={() => refetchDatabases()}
          retryLabel="Retry"
        />
      )}

      {!isPreviewMode && githubServicesError && (
        <ErrorBanner
          className="mb-4"
          message={getErrorMessage(githubServicesError)}
          onRetry={() => refetchGithubServices()}
          retryLabel="Retry"
        />
      )}

      {!isPreviewMode && !isLoading && resources.length === 0 && (
        <div className="rounded-lg border border-dashed border-[var(--color-border-strong)] p-16 text-center text-base text-[var(--color-text-muted)]">
          Nothing deployed yet — spin up a service or provision a database to get started.
        </div>
      )}

      {!isPreviewMode && resources.length > 0 && (
        <>
          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
            {resources.map((resource) =>
              resource.kind === "service" ? (
                <ResourceRow
                  key={`svc-${resource.data.id}`}
                  icon={<Server className="h-5 w-5" />}
                  name={resource.data.name}
                  status={resource.data.status}
                  runtime={resource.data.image}
                  subtitle={`${SERVICE_RESOURCE_LIMITS.cpu} · ${SERVICE_RESOURCE_LIMITS.memory}`}
                  size={`:${resource.data.port}`}
                  domain={resource.data.public_domain}
                  domainHref={
                    resource.data.public_domain ? `https://${resource.data.public_domain}` : undefined
                  }
                  detailHref={`/projects/${projectId}/services/${resource.data.id}`}
                  onUpdate={() =>
                    navigate({
                      to: "/projects/$projectId/services/$serviceId/edit",
                      params: {
                        projectId,
                        serviceId: resource.data.id,
                      },
                    })
                  }
                  onDelete={() => setPendingService(resource.data)}
                  selected={selectedIds.has(resource.data.id)}
                  onToggleSelect={() => toggleSelect(resource.data.id)}
                />
              ) : resource.kind === "github_service" ? (
                <ResourceRow
                  key={`gh-${resource.data.id}`}
                  icon={<Github className="h-5 w-5" />}
                  name={resource.data.name}
                  status={resource.data.status}
                  runtime={resource.data.repo_name}
                  subtitle={`root:${resource.data.root_dir} · ${SERVICE_RESOURCE_LIMITS.cpu} · ${SERVICE_RESOURCE_LIMITS.memory}`}
                  size={`:${resource.data.port}`}
                  domain={resource.data.public_domain}
                  domainHref={
                    resource.data.public_domain ? `https://${resource.data.public_domain}` : undefined
                  }
                  privateDomain={resource.data.private_domain}
                  detailHref={`/projects/${projectId}/github_services/${resource.data.id}`}
                  onUpdate={() =>
                    navigate({
                      to: "/projects/$projectId/github_services/$githubServiceId/edit",
                      params: { projectId, githubServiceId: resource.data.id },
                    })
                  }
                  onDelete={() => setPendingGithubService(resource.data)}
                  selected={selectedIds.has(resource.data.id)}
                  onToggleSelect={() => toggleSelect(resource.data.id)}
                />
              ) : (
                <ResourceRow
                  key={`db-${resource.data.id}`}
                  icon={<DatabaseIcon className="h-5 w-5" />}
                  name={resource.data.name}
                  runtime={resource.data.engine}
                  size={`${resource.data.storage} GB`}
                  domain={resource.data.internal_domain || "internal"}
                  detailHref={`/projects/${projectId}/databases/${resource.data.id}`}
                  onUpdate={() =>
                    navigate({
                      to: "/projects/$projectId/databases/$databaseId/edit",
                      params: {
                        projectId,
                        databaseId: resource.data.id,
                      },
                    })
                  }
                  onDelete={() => setPendingDatabase(resource.data)}
                  selected={selectedIds.has(resource.data.id)}
                  onToggleSelect={() => toggleSelect(resource.data.id)}
                />
              ),
            )}
          </div>

          <ResourceStatusBar
            serviceCount={(services?.length ?? 0) + (githubServices?.length ?? 0)}
            databaseCount={databases?.length ?? 0}
            runningCount={runningCount}
            projectId={projectId}
          />
        </>
      )}

      {isPreviewMode && previewDetail.isLoading && (
        <p className="text-base text-[var(--color-text-faint)]">loading preview…</p>
      )}

      {isPreviewMode && previewDetail.error && (
        <ErrorBanner
          className="mb-4"
          message={getErrorMessage(previewDetail.error)}
          onRetry={() => previewDetail.refetch()}
          retryLabel="Retry"
        />
      )}

      {isPreviewMode && selectedPreviewEnv && (
        <>
          {selectedPreviewEnv.services.length === 0 ? (
            <div className="rounded-lg border border-dashed border-[var(--color-border-strong)] p-16 text-center text-base text-[var(--color-text-muted)]">
              No preview services tracked in this environment yet.
            </div>
          ) : (
            <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
              {selectedPreviewEnv.services.map((s) => {
                const isDb = s.source_type === "database";
                const icon = isDb ? (
                  <DatabaseIcon className="h-5 w-5" />
                ) : s.source_type === "service" ? (
                  <Server className="h-5 w-5" />
                ) : (
                  <Github className="h-5 w-5" />
                );
                return (
                  <ResourceRow
                    key={`preview-${s.preview_env_id}-${s.source_service_id}`}
                    icon={icon}
                    name={s.service_name}
                    status={s.status || undefined}
                    runtime={
                      isDb
                        ? s.engine || "database"
                        : s.source_type === "service"
                          ? "service"
                          : s.repo_name
                    }
                    subtitle={
                      isDb
                        ? "empty copy — no production data"
                        : s.source_type === "service"
                          ? undefined
                          : `${s.branch} · ${s.repo_name}`
                    }
                    size={s.port ? `:${s.port}` : ""}
                    domain={isDb ? "internal" : s.hostname}
                    domainHref={!isDb && s.url ? s.url : undefined}
                    onUpdate={isDb ? undefined : () => setEditingPreview(s)}
                  />
                );
              })}
            </div>
          )}
        </>
      )}

      <DeleteConfirmDialog
        open={!!pendingService}
        onOpenChange={(open) => !open && setPendingService(null)}
        resourceName={pendingService?.name ?? ""}
        resourceLabel="service"
        pending={deleteService.isPending}
        error={deleteService.error ? getErrorMessage(deleteService.error) : undefined}
        onConfirm={() => {
          if (!pendingService) return;
          deleteService.mutate(pendingService.id, {
            onSuccess: () => setPendingService(null),
          });
        }}
      />

      <DeleteConfirmDialog
        open={!!pendingDatabase}
        onOpenChange={(open) => !open && setPendingDatabase(null)}
        resourceName={pendingDatabase?.name ?? ""}
        resourceLabel="database"
        pending={deleteDatabase.isPending}
        error={deleteDatabase.error ? getErrorMessage(deleteDatabase.error) : undefined}
        onConfirm={() => {
          if (!pendingDatabase) return;
          deleteDatabase.mutate(pendingDatabase.id, {
            onSuccess: () => setPendingDatabase(null),
          });
        }}
      />

      <DeleteConfirmDialog
        open={!!pendingGithubService}
        onOpenChange={(open) => !open && setPendingGithubService(null)}
        resourceName={pendingGithubService?.name ?? ""}
        resourceLabel="GitHub service"
        pending={deleteGithubService.isPending}
        error={deleteGithubService.error ? getErrorMessage(deleteGithubService.error) : undefined}
        onConfirm={() => {
          if (!pendingGithubService) return;
          deleteGithubService.mutate(pendingGithubService.id, {
            onSuccess: () => setPendingGithubService(null),
          });
        }}
      />

      <BulkDeleteConfirmDialog
        open={!!pendingBulk}
        onOpenChange={(open) => !open && setPendingBulk(null)}
        resourceNames={pendingBulk?.map((r) => r.data.name) ?? []}
        pending={bulkPending}
        error={bulkError}
        onConfirm={onBulkConfirm}
      />

      <PreviewServiceEditDialog
        projectId={projectId}
        envId={selectedEnv}
        service={editingPreview}
        onClose={() => setEditingPreview(null)}
      />
    </main>
  );
}

function PreviewServiceEditDialog({
  projectId,
  envId,
  service,
  onClose,
}: {
  projectId: string;
  envId: string;
  service: PreviewEnvironment | null;
  onClose: () => void;
}) {
  const update = useUpdatePreviewService(projectId, envId);
  const [envText, setEnvText] = useState("");
  const [portText, setPortText] = useState("");
  const serviceKey = service?.source_service_id ?? "";
  const lastKey = useRef("");
  if (service && lastKey.current !== serviceKey) {
    lastKey.current = serviceKey;
    setEnvText(formatEnvLines(service.env ?? {}));
    setPortText(String(service.port ?? ""));
  }

  const initialEnv = service ? formatEnvLines(service.env ?? {}) : "";
  const initialPort = service ? String(service.port ?? "") : "";
  const portNum = Number(portText);
  const validPort =
    portText.trim() !== "" && Number.isInteger(portNum) && portNum >= 1 && portNum <= 65535;
  const changed = service != null && (envText !== initialEnv || portText !== initialPort);

  return (
    <Dialog
      open={service != null}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
          update.reset();
        }
      }}
    >
      <DialogContent>
        <h3 className="text-lg font-medium text-[var(--color-text)]">
          Update {service?.service_name ?? "preview service"}
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Applies to this preview only — the domain can&apos;t be changed here, and edits
          reset on the next PR sync.
        </p>
        <form
          className="mt-5 space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            if (!service || !changed || !validPort || update.isPending) return;
            const body: { env?: string; port?: number } = {};
            if (envText !== initialEnv) body.env = envText;
            if (portText !== initialPort) body.port = portNum;
            update.mutate(
              { sourceId: service.source_service_id, body },
              { onSuccess: () => onClose() },
            );
          }}
        >
          <div>
            <Label htmlFor="preview-port">Port</Label>
            <Input
              id="preview-port"
              type="number"
              required
              value={portText}
              onChange={(e) => setPortText(e.target.value)}
              className="mt-2 font-mono"
            />
          </div>
          <div>
            <Label htmlFor="preview-env">Environment variables</Label>
            <Textarea
              id="preview-env"
              value={envText}
              onChange={(e) => setEnvText(e.target.value)}
              placeholder={"KEY=value\nANOTHER_KEY=value"}
              rows={6}
              className="mt-2 font-mono"
            />
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              One <code>KEY=value</code> pair per line. Saving replaces the full set of
              environment variables with what&apos;s shown here.
            </p>
          </div>
          {update.error && (
            <ErrorBanner message={getErrorMessage(update.error)} />
          )}
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button size="sm" type="submit" disabled={!changed || !validPort || update.isPending}>
              {update.isPending ? "Saving…" : "Save changes"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
