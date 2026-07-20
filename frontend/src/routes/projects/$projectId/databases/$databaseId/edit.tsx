import { useEffect, useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useDatabase, useUpdateDatabase } from "@/lib/api/databases";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const Route = createFileRoute("/projects/$projectId/databases/$databaseId/edit")({
  component: EditDatabasePage,
});

function EditDatabasePage() {
  const { projectId, databaseId } = Route.useParams();
  const navigate = useNavigate();
  const { data: database } = useDatabase(databaseId);
  const updateDatabase = useUpdateDatabase(projectId, databaseId);

  const [name, setName] = useState("");
  const [storageGB, setStorageGB] = useState("");

  useEffect(() => {
    if (!database) return;
    setName(database.name);
    setStorageGB(String(database.storage));
  }, [database]);

  return (
    <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId/databases/$databaseId"
        params={{ projectId, databaseId }}
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to database
      </Link>
      <h1 className="mt-4 mb-6 font-mono text-2xl font-bold">Update database</h1>

      <form
        className="space-y-5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-6"
        onSubmit={(e) => {
          e.preventDefault();
          updateDatabase.mutate(
            { name, storage_gb: Number(storageGB) },
            {
              onSuccess: () =>
                navigate({
                  to: "/projects/$projectId/databases/$databaseId",
                  params: { projectId, databaseId },
                }),
            },
          );
        }}
      >
        <div>
          <Label htmlFor="name">Database name</Label>
          <Input id="name" required value={name} onChange={(e) => setName(e.target.value)} className="mt-2" />
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

        {updateDatabase.error && (
          <p className="text-sm text-[var(--color-bad)]">{updateDatabase.error.message}</p>
        )}

        <div className="flex gap-3 border-t border-[var(--color-border)] pt-5">
          <Button type="submit" disabled={updateDatabase.isPending}>
            {updateDatabase.isPending ? "Saving…" : "Save changes"}
          </Button>
          <Link
            to="/projects/$projectId/databases/$databaseId"
            params={{ projectId, databaseId }}
          >
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </Link>
        </div>
      </form>
    </main>
  );
}
