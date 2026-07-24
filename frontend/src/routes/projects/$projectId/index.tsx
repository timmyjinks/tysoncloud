import { useMemo, useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { Database as DatabaseIcon, Pencil, Plus, Server } from "lucide-react";
import { useProject } from "@/lib/api/projects";
import { useDeleteService, useServices } from "@/lib/api/services";
import { useDatabases, useDeleteDatabase } from "@/lib/api/databases";
import { ResourceRow } from "@/components/resource-row";
import { ResourceStatusBar } from "@/components/resource-status-bar";
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

  const runningCount = (services ?? []).filter((s) => s.status === "running").length;

  return (
    <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      <div className="flex items-center gap-2">
        <h1 className="font-mono text-3xl font-bold">{project?.name ?? projectId}</h1>
        <Link
          to="/projects/$projectId/edit"
          params={{ projectId }}
          aria-label="Rename project"
          className="text-[var(--color-text-faint)] hover:text-[var(--color-accent)]"
        >
          <Pencil className="h-4 w-4" />
        </Link>
      </div>
      <p className="mt-1 text-base text-[var(--color-text-muted)]">
        Everything deployed in this project
      </p>

      <div className="mt-8 mb-3 flex items-center justify-between">
        <h2 className="text-base font-medium text-[var(--color-text-muted)]">
          Resources <span className="text-[var(--color-text-faint)]">· {resources.length} total</span>
        </h2>
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

      {resources.length > 0 && (
        <>
          <div className="overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
            {resources.map((resource) =>
              resource.kind === "service" ? (
                <ResourceRow
                  key={`svc-${resource.data.id}`}
                  icon={<Server className="h-3.5 w-3.5" />}
                  name={resource.data.name}
                  status={resource.data.status}
                  runtime={resource.data.image}
                  size={`:${resource.data.port}`}
                  domain={resource.data.public_domain}
                  domainHref={
                    resource.data.public_domain ? `https://${resource.data.public_domain}` : undefined
                  }
                  detailHref={`/projects/${projectId}/services/${resource.data.id}`}
                  onDelete={() => setPendingService(resource.data)}
                />
              ) : (
                <ResourceRow
                  key={`db-${resource.data.id}`}
                  icon={<DatabaseIcon className="h-3.5 w-3.5" />}
                  name={resource.data.name}
                  runtime={resource.data.engine}
                  size={`${resource.data.storage} GB`}
                  domain={resource.data.internal_domain || "internal"}
                  detailHref={`/projects/${projectId}/databases/${resource.data.id}`}
                  onDelete={() => setPendingDatabase(resource.data)}
                />
              ),
            )}
          </div>

          <ResourceStatusBar
            serviceCount={services?.length ?? 0}
            databaseCount={databases?.length ?? 0}
            runningCount={runningCount}
            projectId={projectId}
          />
        </>
      )}

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
