import { useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useDeleteService, useService } from "@/lib/api/services";
import { useAttachVolume, useDetachVolume, useVolume } from "@/lib/api/volumes";
import { StatusDot } from "@/components/status-dot";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";

export const Route = createFileRoute("/projects/$projectId/services/$serviceId/")({
  component: ServiceDetail,
});

function ServiceDetail() {
  const { projectId, serviceId } = Route.useParams();
  const navigate = useNavigate();
  const { data: service, isLoading } = useService(serviceId);
  const { data: volume } = useVolume(serviceId);
  const attachVolume = useAttachVolume(projectId, serviceId);
  const detachVolume = useDetachVolume(projectId, serviceId);
  const deleteService = useDeleteService(projectId);

  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [mountPath, setMountPath] = useState("");
  const [storageGB, setStorageGB] = useState("5");

  if (isLoading || !service) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <p className="text-sm text-[var(--color-text-faint)]">loading service…</p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to project
      </Link>

      <div className="mt-4 mb-8 flex items-center gap-4">
        <h1 className="font-mono text-2xl font-bold">{service.name}</h1>
        <div className="flex items-center gap-1.5">
          <StatusDot status={service.status} />
          <span className="text-sm capitalize text-[var(--color-text-muted)]">
            {service.status}
          </span>
        </div>
      </div>

      <section className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Service ID</p>
            <p className="mt-1 font-mono text-sm">{service.id}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Public domain</p>
            <a
              href={`https://${service.public_domain}`}
              target="_blank"
              rel="noreferrer"
              className="mt-1 block font-mono text-sm text-[var(--color-accent)] hover:text-[var(--color-accent-hover)]"
            >
              {service.public_domain}
            </a>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Image</p>
            <p className="mt-1 font-mono text-sm">{service.image}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Port</p>
            <p className="mt-1 font-mono text-sm">{service.port}</p>
          </CardContent>
        </Card>
      </section>

      <section className="mb-8">
        <h2 className="mb-4 text-lg font-semibold">Volume</h2>
        <Card>
          <CardContent className="pt-5">
            {volume ? (
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-mono text-sm">{volume.mount_path}</p>
                  <p className="mt-1 text-xs text-[var(--color-text-faint)]">
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
              <form
                className="flex flex-wrap items-end gap-3"
                onSubmit={(e) => {
                  e.preventDefault();
                  attachVolume.mutate({ mount_path: mountPath, storage_gb: Number(storageGB) });
                }}
              >
                <div className="flex-1 min-w-[160px]">
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
                <div className="w-28">
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
                <Button type="submit" size="sm" disabled={attachVolume.isPending}>
                  {attachVolume.isPending ? "Attaching…" : "Attach volume"}
                </Button>
              </form>
            )}
          </CardContent>
        </Card>
      </section>

      <section className="flex gap-4">
        <Link
          to="/projects/$projectId/services/$serviceId/edit"
          params={{ projectId, serviceId }}
        >
          <Button>Update service</Button>
        </Link>
        <Button variant="danger" onClick={() => setConfirmingDelete(true)}>
          Delete service
        </Button>
      </section>

      <DeleteConfirmDialog
        open={confirmingDelete}
        onOpenChange={setConfirmingDelete}
        resourceName={service.name}
        resourceLabel="service"
        pending={deleteService.isPending}
        error={deleteService.error?.message}
        onConfirm={() =>
          deleteService.mutate(service.id, {
            onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }),
          })
        }
      />
    </main>
  );
}
