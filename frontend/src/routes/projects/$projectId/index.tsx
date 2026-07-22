import { useMemo, useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Database as DatabaseIcon, Plus, Server } from "lucide-react";
import { useProject } from "@/lib/api/projects";
import { useDeleteService, useServices } from "@/lib/api/services";
import { useDatabases, useDeleteDatabase } from "@/lib/api/databases";
import { ResourceCard } from "@/components/resource-card";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";
import { Button } from "@/components/ui/button";
import type { Service, Database } from "@/lib/api/types";

export const Route = createFileRoute("/projects/$projectId/")({
  component: ProjectDetail,
});

type Resource = { kind: "service"; data: Service } | { kind: "database"; data: Database };

function ProjectDetail() {
  const { projectId } = Route.useParams();
  const { data: project } = useProject(projectId);
  const { data: services, isLoading: servicesLoading } = useServices(projectId);
  const { data: databases, isLoading: databasesLoading } = useDatabases(projectId);

  const deleteService = useDeleteService(projectId);
  const deleteDatabase = useDeleteDatabase(projectId);
  const [pendingService, setPendingService] = useState<Service | null>(null);
  const [pendingDatabase, setPendingDatabase] = useState<Database | null>(null);

  const isLoading = servicesLoading || databasesLoading;

  const resources = useMemo<Resource[]>(() => {
    const items: Resource[] = [
      ...(services ?? []).map((s) => ({ kind: "service" as const, data: s })),
      ...(databases ?? []).map((d) => ({ kind: "database" as const, data: d })),
    ];
    return items.sort(
      (a, b) => new Date(b.data.created_at).getTime() - new Date(a.data.created_at).getTime(),
    );
  }, [services, databases]);

  return (
    <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="font-mono text-2xl font-bold">{project?.name ?? projectId}</h1>
      <p className="mt-1 text-sm text-[var(--color-text-muted)]">
        Everything deployed in this project
      </p>

      <div className="mt-8 mb-4 flex items-center justify-between">
        <h2 className="text-lg font-semibold">Resources</h2>
        <div className="flex items-center gap-2">
          <Link to="/projects/$projectId/databases/new" params={{ projectId }}>
            <Button size="sm" variant="outline">
              <Plus className="h-3.5 w-3.5" />
              Database
            </Button>
          </Link>
          <Link to="/projects/$projectId/services/new" params={{ projectId }}>
            <Button size="sm">
              <Plus className="h-3.5 w-3.5" />
              Service
            </Button>
          </Link>
        </div>
      </div>

      {isLoading && (
        <p className="text-sm text-[var(--color-text-faint)]">loading resources…</p>
      )}

      {!isLoading && resources.length === 0 && (
        <div className="rounded-lg border border-dashed border-[var(--color-border-strong)] p-12 text-center text-sm text-[var(--color-text-muted)]">
          Nothing deployed yet — spin up a service or provision a database to get started.
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {resources.map((resource) =>
          resource.kind === "service" ? (
            <ResourceCard
              key={`svc-${resource.data.id}`}
              icon={<Server className="h-3 w-3 text-[var(--color-accent)]" />}
              promptId={resource.data.name}
              title={resource.data.name}
              status={resource.data.status}
              meta={[
                { label: "image", value: resource.data.image, mono: true },
                { label: "port", value: String(resource.data.port), mono: true },
                {
                  label: "domain",
                  value: resource.data.public_domain,
                  mono: true,
                  href: resource.data.public_domain
                    ? `https://${resource.data.public_domain}`
                    : undefined,
                },
              ]}
              detailHref={`/projects/${projectId}/services/${resource.data.id}`}
              onDelete={() => setPendingService(resource.data)}
            />
          ) : (
            <ResourceCard
              key={`db-${resource.data.id}`}
              icon={<DatabaseIcon className="h-3 w-3 text-[var(--color-accent)]" />}
              promptId={`${resource.data.engine} · ${resource.data.name}`}
              title={resource.data.name}
              meta={[
                { label: "engine", value: resource.data.engine, mono: true },
                { label: "storage", value: `${resource.data.storage} GB`, mono: true },
                { label: "host", value: resource.data.internal_domain, mono: true },
              ]}
              detailHref={`/projects/${projectId}/databases/${resource.data.id}`}
              onDelete={() => setPendingDatabase(resource.data)}
            />
          ),
        )}
      </div>

      <DeleteConfirmDialog
        open={!!pendingService}
        onOpenChange={(open) => !open && setPendingService(null)}
        resourceName={pendingService?.name ?? ""}
        resourceLabel="service"
        pending={deleteService.isPending}
        error={deleteService.error?.message}
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
        error={deleteDatabase.error?.message}
        onConfirm={() => {
          if (!pendingDatabase) return;
          deleteDatabase.mutate(pendingDatabase.id, {
            onSuccess: () => setPendingDatabase(null),
          });
        }}
      />
    </main>
  );
}
