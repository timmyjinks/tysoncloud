import { useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useDeleteProject, useProjects } from "@/lib/api/projects";
import { ProjectRow } from "@/components/project-row";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";
import { Button } from "@/components/ui/button";
import type { Project } from "@/lib/api/types";

export const Route = createFileRoute("/dashboard/")({
  component: DashboardIndex,
});

function DashboardIndex() {
  const { data: projects, isLoading, error } = useProjects();
  const deleteProject = useDeleteProject();
  const [pendingDelete, setPendingDelete] = useState<Project | null>(null);

  return (
    <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="font-mono text-3xl font-bold">Projects</h1>
          <p className="text-base text-[var(--color-text-muted)]">
            Everything you're running on TYSONCLOUD
          </p>
        </div>
        <Link to="/dashboard/new">
          <Button>New project</Button>
        </Link>
      </div>

      {isLoading && (
        <p className="text-sm text-[var(--color-text-faint)]">loading projects…</p>
      )}

      {error && (
        <p className="text-sm text-[var(--color-bad)]">
          Couldn't load projects. Try refreshing.
        </p>
      )}

      {projects && projects.length === 0 && (
        <div className="rounded-lg border border-dashed border-[var(--color-border-strong)] p-12 text-center">
          <p className="text-[var(--color-text-muted)]">No projects yet.</p>
          <Link
            to="/dashboard/new"
            className="mt-2 inline-block text-sm font-mono text-[var(--color-accent)] hover:text-[var(--color-accent-hover)]"
          >
            create your first one →
          </Link>
        </div>
      )}

      {projects && projects.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
          {projects.map((project) => (
            <ProjectRow
              key={project.id}
              name={project.name}
              id={project.id}
              href={`/projects/${project.id}`}
              onDelete={() => setPendingDelete(project)}
            />
          ))}
        </div>
      )}

      <DeleteConfirmDialog
        open={!!pendingDelete}
        onOpenChange={(open) => !open && setPendingDelete(null)}
        resourceName={pendingDelete?.name ?? ""}
        resourceLabel="project"
        pending={deleteProject.isPending}
        error={deleteProject.error?.message}
        onConfirm={() => {
          if (!pendingDelete) return;
          deleteProject.mutate(pendingDelete.id, {
            onSuccess: () => setPendingDelete(null),
          });
        }}
      />
    </main>
  );
}
