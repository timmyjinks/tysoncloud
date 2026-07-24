import { useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useCreateDatabase } from "@/lib/api/databases";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";

export const Route = createFileRoute("/projects/$projectId/databases/new")({
  component: NewDatabasePage,
});

function NewDatabasePage() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate();
  const createDatabase = useCreateDatabase(projectId);

  const [name, setName] = useState("");
  const [engine, setEngine] = useState("postgres");
  const [storageGB, setStorageGB] = useState("5");

  return (
    <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to project
      </Link>
      <h1 className="mt-4 mb-6 font-mono text-2xl font-bold">New database</h1>

      <form
        className="space-y-5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-6"
        onSubmit={(e) => {
          e.preventDefault();
          createDatabase.mutate(
            { name, engine, storage_gb: Number(storageGB) },
            { onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }) },
          );
        }}
      >
        <div>
          <Label htmlFor="name">Database name</Label>
          <Input
            id="name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="primary"
            className="mt-2"
          />
        </div>

        <div>
          <Label htmlFor="engine">Engine</Label>
          <Select id="engine" value={engine} onChange={(e) => setEngine(e.target.value)} className="mt-2">
            <option value="postgres">Postgres</option>
          </Select>
        </div>

        <div>
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

        {createDatabase.error && (
          <p className="text-sm text-[var(--color-bad)]">{createDatabase.error.message}</p>
        )}

        <div className="flex gap-3 border-t border-[var(--color-border)] pt-5">
          <Button type="submit" disabled={createDatabase.isPending}>
            {createDatabase.isPending ? "Provisioning…" : "Create database"}
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
