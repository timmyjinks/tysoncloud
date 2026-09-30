import { useState } from "react";
import { MoreVertical, Pencil, Trash2 } from "lucide-react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useDatabase, useDeleteDatabase } from "@/lib/api/databases";
import { getErrorMessage } from "@/lib/api/client";
import { ResourceMetaCard } from "@/components/resource-meta-card";
import { CopyButton } from "@/components/copy-button";
import { cleanEnvValue, formatEnvLines } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";
import { ErrorBanner } from "@/components/error-banner";

export const Route = createFileRoute("/projects/$projectId/databases/$databaseId/")({
  component: DatabaseDetail,
});

function DatabaseDetail() {
  const { projectId, databaseId } = Route.useParams();
  const navigate = useNavigate();
  const { data: database, isLoading, error, refetch } = useDatabase(databaseId);
  const deleteDatabase = useDeleteDatabase(projectId);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  if (error) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
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
      <main className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
        <p className="text-base text-[var(--color-text-faint)]">loading database…</p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="text-base text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to project
      </Link>
      {database.preview && (
        <div className="mt-8 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md bg-[var(--color-surface-2)] px-5 py-3 text-sm text-[var(--color-text-muted)]">
          <span>
            Ephemeral preview {database.preview.name} — empty copy, no production data.
          </span>
          <span className="font-mono text-[var(--color-text-faint)]">
            {database.preview.namespace}
          </span>
        </div>
      )}
      <div className="mt-8 flex flex-wrap items-center justify-between gap-3 border-b border-[var(--color-border)] pb-4">
        <h1 className="font-display text-3xl font-semibold tracking-tight sm:text-4xl">
          {database.name}
        </h1>
        <DropdownMenu
          trigger={
            <Button
              variant="ghost"
              size="icon"
              aria-label="Database actions"
              className="h-9 w-9 text-[var(--color-text-muted)]"
            >
              <MoreVertical className="h-5 w-5" />
            </Button>
          }
        >
          {!database.preview && (
            <>
              <DropdownMenuItem
                onClick={() =>
                  navigate({
                    to: "/projects/$projectId/databases/$databaseId/edit",
                    params: { projectId, databaseId },
                  })
                }
              >
                <Pencil className="h-4 w-4" />
                Update
              </DropdownMenuItem>
              <div className="my-1 h-px bg-[var(--color-border)]" aria-hidden="true" />
            </>
          )}
          <DropdownMenuItem destructive onClick={() => setConfirmingDelete(true)}>
            <Trash2 className="h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenu>
      </div>

      <div className="mt-6">
        <ResourceMetaCard
          meta={[
            { label: "Database ID", value: database.id, mono: true },
            { label: "Engine", value: database.engine, mono: true },
            { label: "Internal host", value: database.internal_domain, mono: true, copyable: true, danger: true },
            { label: "Port", value: String(database.port), mono: true },
            { label: "Storage", value: `${database.storage} GB`, mono: true },
          ]}
        />
      </div>

      <section className="mt-8">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-display text-lg font-semibold text-[var(--color-text)]">
            Environment variables
          </h2>
          {Object.keys(database.env ?? {}).length > 0 && (
            <CopyButton
              label="Copy all environment variables"
              value={formatEnvLines(database.env)}
            />
          )}
        </div>
        {Object.keys(database.env ?? {}).length > 0 ? (
          <div className="overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
            {Object.entries(database.env).map(([key, value]) => (
              <CopyButton
                key={key}
                label={`Copy ${key}=${value}`}
                value={`${key}=${cleanEnvValue(value)}`}
                className="border-t border-[var(--color-border)] first:border-t-0"
              >
                <span className="text-[var(--color-text)]">{key}</span>
                <span className="text-[var(--color-text-faint)]">={cleanEnvValue(value)}</span>
              </CopyButton>
            ))}
          </div>
        ) : (
          <p className="text-base text-[var(--color-text-faint)]">
            No environment variables set.
          </p>
        )}
      </section>

      <DeleteConfirmDialog
        open={confirmingDelete}
        onOpenChange={setConfirmingDelete}
        resourceName={database.name}
        resourceLabel="database"
        pending={deleteDatabase.isPending}
        error={deleteDatabase.error ? getErrorMessage(deleteDatabase.error) : undefined}
        onConfirm={() =>
          deleteDatabase.mutate(database.id, {
            onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }),
          })
        }
      />
    </main>
  );
}