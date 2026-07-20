import { useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useDatabase, useDeleteDatabase } from "@/lib/api/databases";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";

export const Route = createFileRoute("/projects/$projectId/databases/$databaseId/")({
  component: DatabaseDetail,
});

function DatabaseDetail() {
  const { projectId, databaseId } = Route.useParams();
  const navigate = useNavigate();
  const { data: database, isLoading } = useDatabase(databaseId);
  const deleteDatabase = useDeleteDatabase(projectId);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  if (isLoading || !database) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <p className="text-sm text-[var(--color-text-faint)]">loading database…</p>
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
      <h1 className="mt-4 mb-8 font-mono text-2xl font-bold">{database.name}</h1>

      <section className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Database ID</p>
            <p className="mt-1 font-mono text-sm">{database.id}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Engine</p>
            <p className="mt-1 font-mono text-sm capitalize">{database.engine}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Internal host</p>
            <p className="mt-1 font-mono text-sm">{database.internal_domain}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Port</p>
            <p className="mt-1 font-mono text-sm">{database.port}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5">
            <p className="text-sm text-[var(--color-text-faint)]">Storage</p>
            <p className="mt-1 font-mono text-sm">{database.storage} GB</p>
          </CardContent>
        </Card>
      </section>

      <section className="flex gap-4">
        <Link
          to="/projects/$projectId/databases/$databaseId/edit"
          params={{ projectId, databaseId }}
        >
          <Button>Update database</Button>
        </Link>
        <Button variant="danger" onClick={() => setConfirmingDelete(true)}>
          Delete database
        </Button>
      </section>

      <DeleteConfirmDialog
        open={confirmingDelete}
        onOpenChange={setConfirmingDelete}
        resourceName={database.name}
        resourceLabel="database"
        pending={deleteDatabase.isPending}
        error={deleteDatabase.error?.message}
        onConfirm={() =>
          deleteDatabase.mutate(database.id, {
            onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }),
          })
        }
      />
    </main>
  );
}
