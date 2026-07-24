import { useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useCreateProject } from "@/lib/api/projects";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const Route = createFileRoute("/dashboard/new")({
  component: NewProjectPage,
});

function NewProjectPage() {
  const navigate = useNavigate();
  const createProject = useCreateProject();
  const [name, setName] = useState("");

  return (
    <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
      <Link
        to="/dashboard"
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to dashboard
      </Link>
      <h1 className="mt-4 mb-6 font-mono text-2xl font-bold">New project</h1>

      <form
        className="space-y-5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-6"
        onSubmit={(e) => {
          e.preventDefault();
          createProject.mutate(
            { name },
            { onSuccess: () => navigate({ to: "/dashboard" }) },
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
            placeholder="my-app"
            className="mt-2"
          />
        </div>

        {createProject.error && (
          <p className="text-sm text-[var(--color-bad)]">{createProject.error.message}</p>
        )}

        <div className="flex gap-3 border-t border-[var(--color-border)] pt-5">
          <Button type="submit" disabled={createProject.isPending}>
            {createProject.isPending ? "Creating…" : "Create project"}
          </Button>
          <Link to="/dashboard">
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </Link>
        </div>
      </form>
    </main>
  );
}
