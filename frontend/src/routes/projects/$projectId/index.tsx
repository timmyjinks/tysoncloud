import { useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
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

function ProjectDetail() {
  const { projectId } = Route.useParams();
  const { data: project } = useProject(projectId);
  const { data: services, isLoading: servicesLoading } = useServices(projectId);
  const { data: databases, isLoading: databasesLoading } = useDatabases(projectId);

  const deleteService = useDeleteService(projectId);
  const deleteDatabase = useDeleteDatabase(projectId);
  const [pendingService, setPendingService] = useState<Service | null>(null);
  const [pendingDatabase, setPendingDatabase] = useState<Database | null>(null);

  return (
    <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <Link
        to="/dashboard"
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to dashboard
      </Link>
      <h1 className="mt-4 mb-8 font-mono text-2xl font-bold">
        {project?.name ?? projectId}
      </h1>

      <section className="mb-12">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">Services</h2>
          <Link to="/projects/$projectId/services/new" params={{ projectId }}>
            <Button size="sm">New service</Button>
          </Link>
        </div>

        {servicesLoading && (
          <p className="text-sm text-[var(--color-text-faint)]">loading services…</p>
        )}

        {services && services.length === 0 && (
          <div className="rounded-lg border border-dashed border-[var(--color-border-strong)] p-8 text-center text-sm text-[var(--color-text-muted)]">
            No services deployed yet.
          </div>
        )}

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {services?.map((service) => (
            <ResourceCard
              key={service.id}
              promptId={service.name}
              title={service.name}
              status={service.status}
              meta={[
                { label: "image", value: service.image, mono: true },
                { label: "port", value: String(service.port), mono: true },
                {
                  label: "domain",
                  value: service.public_domain,
                  mono: true,
                  href: service.public_domain ? `https://${service.public_domain}` : undefined,
                },
              ]}
              detailHref={`/projects/${projectId}/services/${service.id}`}
              onDelete={() => setPendingService(service)}
            />
          ))}
        </div>
      </section>

      <section>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">Databases</h2>
          <Link to="/projects/$projectId/databases/new" params={{ projectId }}>
            <Button size="sm">New database</Button>
          </Link>
        </div>

        {databasesLoading && (
          <p className="text-sm text-[var(--color-text-faint)]">loading databases…</p>
        )}

        {databases && databases.length === 0 && (
          <div className="rounded-lg border border-dashed border-[var(--color-border-strong)] p-8 text-center text-sm text-[var(--color-text-muted)]">
            No databases provisioned yet.
          </div>
        )}

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {databases?.map((database) => (
            <ResourceCard
              key={database.id}
              promptId={`${database.engine} · ${database.name}`}
              title={database.name}
              meta={[
                { label: "engine", value: database.engine, mono: true },
                { label: "storage", value: `${database.storage} GB`, mono: true },
                { label: "host", value: database.internal_domain, mono: true },
              ]}
              detailHref={`/projects/${projectId}/databases/${database.id}`}
              onDelete={() => setPendingDatabase(database)}
            />
          ))}
        </div>
      </section>

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
