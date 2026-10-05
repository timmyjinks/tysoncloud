import { useEffect, useState } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useDatabase, useUpdateDatabase } from "@/lib/api/databases";
import { getErrorMessage } from "@/lib/api/client";
import { ErrorBanner } from "@/components/error-banner";
import { FormShell } from "@/components/form-shell";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const Route = createFileRoute("/projects/$projectId/databases/$databaseId/edit")({
  component: EditDatabasePage,
});

function EditDatabasePage() {
  const { projectId, databaseId } = Route.useParams();
  const navigate = useNavigate();
  const { data: database, isLoading, error, refetch } = useDatabase(databaseId);
  const updateDatabase = useUpdateDatabase(projectId, databaseId);

  const [name, setName] = useState("");
  const [storageGB, setStorageGB] = useState("");

  useEffect(() => {
    if (!database) return;
    setName(database.name);
    setStorageGB(String(database.storage));
  }, [database]);

  if (error) {
    return (
      <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
        <ErrorBanner
          message={getErrorMessage(error)}
          onRetry={() => refetch()}
          retryLabel="Retry"
        />
      </main>
    );
  }

  if (isLoading || !database) {
    return (
      <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
        <p className="text-sm text-[var(--color-text-faint)]">loading database…</p>
      </main>
    );
  }

  return (
    <FormShell
      backTo="/projects/$projectId/databases/$databaseId"
      backLabel="Back to database"
      title="Update database"
      onSubmit={(e) => {
        e.preventDefault();
        updateDatabase.mutate(
          { name, engine: database?.engine, storage_gb: Number(storageGB) },
          {
            onSuccess: () =>
              navigate({
                to: "/projects/$projectId/databases/$databaseId",
                params: { projectId, databaseId },
                search: { env: undefined },
              }),
          },
        );
      }}
      error={updateDatabase.error ? getErrorMessage(updateDatabase.error) : undefined}
      pending={updateDatabase.isPending}
      submitLabel="Save changes"
      pendingLabel="Saving…"
      cancelTo="/projects/$projectId/databases/$databaseId"
    >
      <div>
        <Label htmlFor="name">Database name</Label>
        <Input
          id="name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="mt-2"
        />
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
    </FormShell>
  );
}