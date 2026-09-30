import { useState } from "react";
import { MoreVertical, Pencil, RefreshCw, Terminal, Trash2 } from "lucide-react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useDeleteGithubService, useGithubService, useRedeployGithubService } from "@/lib/api/github";
import {
  useDeletePreviewService,
  useProjectPreviewEnvironment,
} from "@/lib/api/previews";
import { getErrorMessage } from "@/lib/api/client";
import { ResourceMetaCard } from "@/components/resource-meta-card";
import { CopyButton } from "@/components/copy-button";
import { cleanEnvValue, formatEnvLines } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";
import { ErrorBanner } from "@/components/error-banner";
import { GithubServiceLogsDrawer } from "@/components/github-service-logs-drawer";
import { PreviewServiceEditDialog } from "@/components/preview-service-edit-dialog";
import { SERVICE_RESOURCE_LIMITS } from "@/lib/resource-limits";

export const Route = createFileRoute("/projects/$projectId/github_services/$githubServiceId/")({
  validateSearch: (search: Record<string, unknown>) => ({
    env: typeof search.env === "string" ? search.env : undefined,
  }),
  component: GithubServiceDetail,
});

function GithubServiceDetail() {
  const { projectId, githubServiceId } = Route.useParams();
  const { env: previewEnvId } = Route.useSearch();
  const navigate = useNavigate();
  // Preview copies render from the preview detail API, never the prod
  // endpoints — the prod handlers are prod-only.
  const previewDetail = useProjectPreviewEnvironment(projectId, previewEnvId ?? null);
  const previewEntry = previewDetail.data?.services.find(
    (s) => s.source_service_id === githubServiceId,
  );

  const { data: service, isLoading, error, refetch } = useGithubService(githubServiceId, {
    enabled: !previewEnvId,
  });
  const deleteGithubService = useDeleteGithubService(projectId);
  const deletePreview = useDeletePreviewService(projectId, previewEnvId ?? "");
  const redeployGithubService = useRedeployGithubService(projectId, githubServiceId);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [logsOpen, setLogsOpen] = useState(false);
  const [editingPreview, setEditingPreview] = useState(false);
  const isDeploying =
    service?.status === "building" || service?.status === "deploying" || redeployGithubService.isPending;

  if (previewEnvId) {
    if (previewDetail.isLoading) {
      return (
        <main className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
          <p className="text-base text-[var(--color-text-faint)]">loading preview…</p>
        </main>
      );
    }
    if (previewDetail.error || !previewEntry) {
      return (
        <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
          <ErrorBanner
            message={
              previewDetail.error
                ? getErrorMessage(previewDetail.error)
                : "We couldn't find that service in this preview environment."
            }
            onRetry={() => previewDetail.refetch()}
            retryLabel="Retry"
          />
        </main>
      );
    }

    return (
      <main className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
        <Link
          to="/projects/$projectId"
          params={{ projectId }}
          className="text-base text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
        >
          ← Back to project
        </Link>

        <div className="mt-8 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md bg-[var(--color-surface-2)] px-5 py-3 text-sm text-[var(--color-text-muted)]">
          <span>
            Ephemeral preview {previewEntry.env_name} — managed by the pull request. Edits
            apply here only and reset on the next sync.
          </span>
          {previewEntry.pr_url && (
            <a
              href={previewEntry.pr_url}
              target="_blank"
              rel="noreferrer"
              className="font-mono text-[var(--color-accent)] hover:text-[var(--color-accent-hover)]"
            >
              #{previewEntry.pr} ↗
            </a>
          )}
          <span className="font-mono text-[var(--color-text-faint)]">
            {previewDetail.data?.namespace}
          </span>
        </div>

        <div className="mt-8 flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-border)] pb-4">
          <h1 className="font-display text-3xl font-semibold tracking-tight sm:text-4xl">{previewEntry.service_name}</h1>
          <DropdownMenu
            trigger={
              <Button
                variant="ghost"
                size="icon"
                aria-label="Service actions"
                className="h-9 w-9 text-[var(--color-text-muted)]"
              >
                <MoreVertical className="h-5 w-5" />
              </Button>
            }
          >
            <DropdownMenuItem onClick={() => setLogsOpen(true)}>
              <Terminal className="h-4 w-4" />
              View logs
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setEditingPreview(true)}>
              <Pencil className="h-4 w-4" />
              Update
            </DropdownMenuItem>
            <div className="my-1 h-px bg-[var(--color-border)]" aria-hidden="true" />
            <DropdownMenuItem destructive onClick={() => setConfirmingDelete(true)}>
              <Trash2 className="h-4 w-4" />
              Delete
            </DropdownMenuItem>
          </DropdownMenu>
        </div>

        <div className="mt-4">
          <ResourceMetaCard
            meta={[
              { label: "Status", value: previewEntry.status, status: true },
              { label: "Service ID", value: previewEntry.source_service_id, mono: true },
              {
                label: "Repository",
                value: previewEntry.repo_name,
                mono: true,
                href: `https://github.com/${previewEntry.repo_name}`,
              },
              { label: "Branch", value: previewEntry.branch || "main", mono: true },
              {
                label: "Public domain",
                value: previewEntry.hostname,
                mono: true,
                href: previewEntry.url || undefined,
              },
              { label: "Port", value: String(previewEntry.port), mono: true },
              { label: "Max CPU", value: SERVICE_RESOURCE_LIMITS.cpu, mono: true },
              { label: "Max memory", value: SERVICE_RESOURCE_LIMITS.memory, mono: true },
            ]}
          />
        </div>

        <section className="mt-8">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="font-display text-lg font-semibold text-[var(--color-text)]">Environment variables</h2>
            {Object.keys(previewEntry.env ?? {}).length > 0 && (
              <CopyButton label="Copy all environment variables" value={formatEnvLines(previewEntry.env)} />
            )}
          </div>
          {Object.keys(previewEntry.env ?? {}).length > 0 ? (
            <div className="overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
              {Object.entries(previewEntry.env).map(([key, value]) => (
                <CopyButton
                  key={key}
                  label={`Copy ${key}=${value}`}
                  value={`${key}=${cleanEnvValue(value)}`}
                  className="border-t border-[var(--color-border)] first:border-t-0"
                >
                  <span className="text-[var(--color-text)]">{key}</span>
                  <span className="text-[var(--color-text-faint)]">={cleanEnvValue(value)}</span>
                </CopyButton>
              ))}
            </div>
          ) : (
            <p className="text-base text-[var(--color-text-faint)]">No environment variables set.</p>
          )}
        </section>

        <DeleteConfirmDialog
          open={confirmingDelete}
          onOpenChange={setConfirmingDelete}
          resourceName={previewEntry.service_name}
          resourceLabel="preview service"
          pending={deletePreview.isPending}
          error={deletePreview.error ? getErrorMessage(deletePreview.error) : undefined}
          onConfirm={() =>
            deletePreview.mutate(previewEntry.source_service_id, {
              onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }),
            })
          }
        />

        <GithubServiceLogsDrawer
          open={logsOpen}
          onOpenChange={setLogsOpen}
          projectId={projectId}
          githubServiceId={previewEntry.source_service_id}
          serviceName={previewEntry.service_name}
          preview={{ envId: previewEnvId, sourceId: previewEntry.source_service_id }}
        />

        <PreviewServiceEditDialog
          projectId={projectId}
          envId={previewEnvId}
          service={
            editingPreview
              ? {
                  sourceId: previewEntry.source_service_id,
                  serviceName: previewEntry.service_name,
                  env: previewEntry.env ?? {},
                  port: previewEntry.port,
                }
              : null
          }
          onClose={() => setEditingPreview(false)}
          onSaved={() => previewDetail.refetch()}
        />
      </main>
    );
  }

  if (error) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <ErrorBanner message={getErrorMessage(error)} onRetry={() => refetch()} retryLabel="Retry" />
      </main>
    );
  }

  if (isLoading || !service) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
        <p className="text-base text-[var(--color-text-faint)]">loading service…</p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="text-base text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to project
      </Link>

      <div className="mt-8 flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-border)] pb-4">
        <h1 className="font-display text-3xl font-semibold tracking-tight sm:text-4xl">{service.name}</h1>
        <DropdownMenu
          trigger={
            <Button
              variant="ghost"
              size="icon"
              aria-label="Service actions"
              className="h-9 w-9 text-[var(--color-text-muted)]"
            >
              <MoreVertical className="h-5 w-5" />
            </Button>
          }
        >
          <DropdownMenuItem onClick={() => setLogsOpen(true)}>
            <Terminal className="h-4 w-4" />
            View logs
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => {
              if (isDeploying) return;
              redeployGithubService.mutate(undefined, {
                onSuccess: () => setLogsOpen(true),
              });
            }}
          >
            <RefreshCw className="h-4 w-4" />
            {isDeploying ? "Redeploying…" : "Redeploy"}
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() =>
              navigate({
                to: "/projects/$projectId/github_services/$githubServiceId/edit",
                params: { projectId, githubServiceId },
              })
            }
          >
            <Pencil className="h-4 w-4" />
            Update
          </DropdownMenuItem>
          <div className="my-1 h-px bg-[var(--color-border)]" aria-hidden="true" />
          <DropdownMenuItem destructive onClick={() => setConfirmingDelete(true)}>
            <Trash2 className="h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenu>
      </div>

      <div className="mt-4">
        {redeployGithubService.error && (
          <div className="mb-4">
            <ErrorBanner message={getErrorMessage(redeployGithubService.error)} />
          </div>
        )}
        <ResourceMetaCard
          meta={[
            { label: "Status", value: service.status, status: true },
            { label: "Service ID", value: service.id, mono: true },
            {
              label: "Repository",
              value: service.repo_name,
              mono: true,
              href: `https://github.com/${service.repo_name}`,
            },
            { label: "Branch", value: service.branch || "main", mono: true },
            { label: "Root directory", value: service.root_dir || ".", mono: true },
            {
              label: "Public domain",
              value: service.public_domain,
              mono: true,
              href: `https://${service.public_domain}`,
            },
            {
              label: "Private domain",
              value: service.private_domain,
              mono: true,
              copyable: true,
            },
            { label: "Port", value: String(service.port), mono: true },
            { label: "Max CPU", value: SERVICE_RESOURCE_LIMITS.cpu, mono: true },
            { label: "Max memory", value: SERVICE_RESOURCE_LIMITS.memory, mono: true },
          ]}
        />
      </div>

      <section className="mt-8">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-display text-lg font-semibold text-[var(--color-text)]">Environment variables</h2>
          {Object.keys(service.env ?? {}).length > 0 && (
            <CopyButton label="Copy all environment variables" value={formatEnvLines(service.env)} />
          )}
        </div>
        {Object.keys(service.env ?? {}).length > 0 ? (
          <div className="overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
            {Object.entries(service.env).map(([key, value]) => (
              <CopyButton
                key={key}
                label={`Copy ${key}=${value}`}
                value={`${key}=${cleanEnvValue(value)}`}
                className="border-t border-[var(--color-border)] first:border-t-0"
              >
                <span className="text-[var(--color-text)]">{key}</span>
                <span className="text-[var(--color-text-faint)]">={cleanEnvValue(value)}</span>
              </CopyButton>
            ))}
          </div>
        ) : (
          <p className="text-base text-[var(--color-text-faint)]">No environment variables set.</p>
        )}
      </section>

      <DeleteConfirmDialog
        open={confirmingDelete}
        onOpenChange={setConfirmingDelete}
        resourceName={service.name}
        resourceLabel="service"
        pending={deleteGithubService.isPending}
        error={deleteGithubService.error ? getErrorMessage(deleteGithubService.error) : undefined}
        onConfirm={() =>
          deleteGithubService.mutate(service.id, {
            onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }),
          })
        }
      />

      <GithubServiceLogsDrawer
        open={logsOpen}
        onOpenChange={setLogsOpen}
        projectId={projectId}
        githubServiceId={service.id}
        serviceName={service.name}
      />
    </main>
  );
}
