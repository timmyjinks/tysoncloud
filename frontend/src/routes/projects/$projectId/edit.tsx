import { useEffect, useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useProject, useUpdateProject } from "@/lib/api/projects";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const Route = createFileRoute("/projects/$projectId/edit")({
  component: EditProjectPage,
});

function EditProjectPage() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate();
  const { data: project } = useProject(projectId);
  const updateProject = useUpdateProject(projectId);

  const [name, setName] = useState("");

  useEffect(() => {
    if (!project) return;
    setName(project.name);
  }, [project]);

  return (
    <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to project
      </Link>
      <h1 className="mt-4 mb-6 font-mono text-3xl font-bold">Rename project</h1>

      <form
        className="space-y-5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-6"
        onSubmit={(e) => {
          e.preventDefault();
          updateProject.mutate(
            { name },
            {
              onSuccess: () =>
                navigate({ to: "/projects/$projectId", params: { projectId } }),
            },
          );
        }}
      >
        <div>
          <Label htmlFor="name">Project name</Label>
          <Input
            id="name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="mt-2"
          />
        </div>

        {updateProject.error && (
          <p className="text-sm text-[var(--color-bad)]">{updateProject.error.message}</p>
        )}

        <div className="flex gap-3 border-t border-[var(--color-border)] pt-5">
          <Button type="submit" disabled={updateProject.isPending}>
            {updateProject.isPending ? "Saving…" : "Save changes"}
          </Button>
          <Link to="/projects/$projectId" params={{ projectId }}>
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </Link>
        </div>
      </form>
    </main>
  );
}
