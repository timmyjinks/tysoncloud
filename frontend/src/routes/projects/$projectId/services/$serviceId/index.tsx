import { useState } from "react";
import { MoreVertical, Pencil, Terminal, Trash2 } from "lucide-react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useDeleteService, useService } from "@/lib/api/services";
import { useAttachVolume, useDetachVolume, useVolume } from "@/lib/api/volumes";
import { getErrorMessage } from "@/lib/api/client";
import { ResourceMetaCard } from "@/components/resource-meta-card";
import { CopyButton } from "@/components/copy-button";
import { cleanEnvValue, formatEnvLines } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";
import { ServiceLogsDrawer } from "@/components/service-logs-drawer";
import { ErrorBanner } from "@/components/error-banner";
import { PreviewServiceEditDialog } from "@/components/preview-service-edit-dialog";
import { SERVICE_RESOURCE_LIMITS } from "@/lib/resource-limits";

export const Route = createFileRoute("/projects/$projectId/services/$serviceId/")({
  component: ServiceDetail,
});

function ServiceDetail() {
  const { projectId, serviceId } = Route.useParams();
  const navigate = useNavigate();
  const { data: service, isLoading, error, refetch } = useService(serviceId);
  const {
    data: volume,
    error: volumeError,
    refetch: refetchVolume,
  } = useVolume(serviceId);
  const attachVolume = useAttachVolume(projectId, serviceId);
  const detachVolume = useDetachVolume(projectId, serviceId);
  const deleteService = useDeleteService(projectId);

  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [mountPath, setMountPath] = useState("");
  const [storageGB, setStorageGB] = useState("5");
  const [logsOpen, setLogsOpen] = useState(false);
  const [editingPreview, setEditingPreview] = useState(false);

  if (error) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <ErrorBanner
          message={getErrorMessage(error)}
          onRetry={() => refetch()}
          retryLabel="Retry"
        />
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

      {service.preview && (
        <div className="mt-8 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md bg-[var(--color-surface-2)] px-5 py-3 text-sm text-[var(--color-text-muted)]">
          <span>
            Ephemeral preview {service.preview.name} — managed by the pull request. Edits
            apply here only and reset on the next sync.
          </span>
          <span className="font-mono text-[var(--color-text-faint)]">
            {service.preview.namespace}
          </span>
        </div>
      )}

      <div className="mt-8 flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-border)] pb-4">
        <h1 className="font-display text-3xl font-semibold tracking-tight sm:text-4xl">
          {service.name}
        </h1>
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
            onClick={() =>
              service.preview
                ? setEditingPreview(true)
                : navigate({
                    to: "/projects/$projectId/services/$serviceId/edit",
                    params: { projectId, serviceId },
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
        <ResourceMetaCard
          meta={[
            { label: "Status", value: service.status, status: true },
            { label: "Service ID", value: service.id, mono: true },
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
            { label: "Image", value: service.image, mono: true, copyable: true },
            { label: "Port", value: String(service.port), mono: true },
            { label: "Max CPU", value: SERVICE_RESOURCE_LIMITS.cpu, mono: true },
            { label: "Max memory", value: SERVICE_RESOURCE_LIMITS.memory, mono: true },
          ]}
        />
      </div>

      <section className="mt-8">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-display text-lg font-semibold text-[var(--color-text)]">
            Environment variables
          </h2>
          {Object.keys(service.env ?? {}).length > 0 && (
            <CopyButton
              label="Copy all environment variables"
              value={formatEnvLines(service.env)}
            />
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
          <p className="text-base text-[var(--color-text-faint)]">
            No environment variables set.
          </p>
        )}
      </section>

      {!service.preview && (
      <section className="mt-8">
        <h2 className="mb-3 font-display text-lg font-semibold text-[var(--color-text)]">
          Volume
        </h2>
        <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
          {volumeError && (
            <ErrorBanner
              className="mb-3"
              message={getErrorMessage(volumeError)}
              onRetry={() => refetchVolume()}
              retryLabel="Retry"
            />
          )}
          {volume ? (
            <div className="flex items-center justify-between px-5 py-4">
              <div>
                <p className="font-mono text-base">{volume.mount_path}</p>
                <p className="mt-1 text-base text-[var(--color-text-faint)]">
                  {volume.storage_gb} GB
                </p>
              </div>
              <Button
                variant="danger"
                size="sm"
                disabled={detachVolume.isPending}
                onClick={() => detachVolume.mutate()}
              >
                {detachVolume.isPending ? "Detaching…" : "Detach"}
              </Button>
            </div>
          ) : (
            <>
              <form
                className="flex flex-wrap items-end gap-3 px-5 py-4"
                onSubmit={(e) => {
                  e.preventDefault();
                  attachVolume.mutate({ mount_path: mountPath, storage_gb: Number(storageGB) });
                }}
              >
                <div className="flex-1 min-w-[180px]">
                  <Label htmlFor="mount_path">Mount path</Label>
                  <Input
                    id="mount_path"
                    required
                    value={mountPath}
                    onChange={(e) => setMountPath(e.target.value)}
                    placeholder="/app/data"
                    className="mt-2 font-mono"
                  />
                </div>
                <div className="w-32">
                  <Label htmlFor="storage_gb">Storage (GB)</Label>
                  <Input
                    id="storage_gb"
                    type="number"
                    required
                    value={storageGB}
                    onChange={(e) => setStorageGB(e.target.value)}
                    className="mt-2 font-mono"
                  />
                </div>
                <Button type="submit" disabled={attachVolume.isPending}>
                  {attachVolume.isPending ? "Attaching…" : "Attach volume"}
                </Button>
              </form>
              {attachVolume.error && (
                <ErrorBanner className="mt-3" message={getErrorMessage(attachVolume.error)} />
              )}
            </>
          )}
          {detachVolume.error && (
            <ErrorBanner className="mt-3" message={getErrorMessage(detachVolume.error)} />
          )}
        </div>
      </section>
      )}

      <DeleteConfirmDialog
        open={confirmingDelete}
        onOpenChange={setConfirmingDelete}
        resourceName={service.name}
        resourceLabel={service.preview ? "preview service" : "service"}
        pending={deleteService.isPending}
        error={deleteService.error ? getErrorMessage(deleteService.error) : undefined}
        onConfirm={() =>
          deleteService.mutate(service.id, {
            onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }),
          })
        }
      />

      <ServiceLogsDrawer
        open={logsOpen}
        onOpenChange={setLogsOpen}
        projectId={projectId}
        serviceId={service.id}
        serviceName={service.name}
      />

      {service.preview && (
        <PreviewServiceEditDialog
          projectId={projectId}
          envId={service.preview.env_id}
          service={
            editingPreview
              ? {
                  sourceId: service.id,
                  serviceName: service.name,
                  env: service.env ?? {},
                  port: service.port,
                }
              : null
          }
          onClose={() => setEditingPreview(false)}
          onSaved={() => refetch()}
        />
      )}
    </main>
  );
}