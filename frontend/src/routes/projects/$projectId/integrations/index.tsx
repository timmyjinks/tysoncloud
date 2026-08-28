import { useEffect, useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Github, ExternalLink, Trash2 } from "lucide-react";
import {
  useCreateGithubConnection,
  useDeleteGithubConnection,
  useGithubApp,
  useGithubConnections,
  useGithubRepos,
} from "@/lib/api/github";
import { getErrorMessage } from "@/lib/api/client";
import { ErrorBanner } from "@/components/error-banner";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DeleteConfirmDialog } from "@/components/delete-confirm-dialog";

export const Route = createFileRoute("/projects/$projectId/integrations/")({
  validateSearch: (search: Record<string, unknown>) => ({
    installation_id: typeof search.installation_id === "string" ? search.installation_id : undefined,
    setup_action: typeof search.setup_action === "string" ? search.setup_action : undefined,
  }),
  component: IntegrationsPage,
});

function IntegrationsPage() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate();
  const { installation_id: installationIdFromQuery } = Route.useSearch();
  const { data: connections, isLoading, error, refetch } = useGithubConnections();
  const createConnection = useCreateGithubConnection();
  const deleteConnection = useDeleteGithubConnection();
  const [installationIdInput, setInstallationIdInput] = useState("");
  const [confirmingDeleteId, setConfirmingDeleteId] = useState<string | null>(null);

  const connection = connections?.[0] ?? null;
  const installationId = connection?.installation_id ?? installationIdFromQuery ?? "";
  const { data: reposData } = useGithubRepos(installationId);

  // Auto-create connection if redirected back with installation_id and no existing connection.
  const [autoCreated, setAutoCreated] = useState(false);
  useEffect(() => {
    if (installationIdFromQuery && !isLoading && !connection && !autoCreated && !createConnection.isPending) {
      setAutoCreated(true);
      createConnection.mutate(
        { installation_id: installationIdFromQuery },
        {
          onSuccess: () => {
            navigate({ to: "/projects/$projectId/integrations", params: { projectId }, search: {} as never });
          },
        },
      );
    }
  }, [installationIdFromQuery, isLoading, connection, autoCreated, createConnection, navigate, projectId]);

  const { data: appInfo, isLoading: appLoading } = useGithubApp();
  const installUrl = appInfo?.install_url || "";
  // GitHub Apps pass `state` through to the Setup URL (configured as /github/callback).
  // We use projectId as state so the callback can redirect back to the right project.
  const installUrlWithState = installUrl
    ? `${installUrl}${installUrl.includes("?") ? "&" : "?"}state=${encodeURIComponent(projectId)}`
    : "";

  const handleInstall = () => {
    if (!installUrlWithState) return;
    const w = 600;
    const h = 700;
    const left = window.screenX + (window.outerWidth - w) / 2;
    const top = window.screenY + (window.outerHeight - h) / 2;
    const popup = window.open(
      installUrlWithState,
      "tysoncloud-github-install",
      `popup=yes,width=${w},height=${h},left=${left},top=${top},scrollbars=yes`,
    );
    // Popup blocked → fall back to full redirect
    if (!popup) {
      window.location.href = installUrlWithState;
      return;
    }
    // Poll for popup close (GitHub redirects to /github/callback inside popup which then closes itself)
    const timer = window.setInterval(() => {
      if (popup.closed) {
        window.clearInterval(timer);
        refetch();
      }
    }, 500);
  };

  // Listen for postMessage from /github/callback when it runs inside the popup
  useEffect(() => {
    const handler = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;
      if (event.data?.type === "github-app-installed") {
        refetch();
      }
    };
    window.addEventListener("message", handler);
    return () => window.removeEventListener("message", handler);
  }, [refetch]);

  const confirmingConnection = connections?.find((c) => c.id === confirmingDeleteId) ?? null;

  return (
    <main className="mx-auto max-w-4xl px-4 py-10 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="text-base text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to project
      </Link>

      <div className="mt-8">
        <PageHeader title="Integrations" description="Connect GitHub to deploy from repositories." />
      </div>

      {error && (
        <ErrorBanner className="mt-6" message={getErrorMessage(error)} onRetry={() => refetch()} retryLabel="Retry" />
      )}

      {isLoading ? (
        <p className="mt-6 text-base text-[var(--color-text-faint)]">loading integrations…</p>
      ) : connection ? (
        <div className="mt-6 space-y-6">
          <section className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-6">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-center gap-3">
                <span className="flex h-10 w-10 items-center justify-center rounded-md bg-[var(--color-surface-2)]">
                  <Github className="h-5 w-5" />
                </span>
                <div>
                  <h3 className="font-medium text-[var(--color-text)]">GitHub</h3>
                  <p className="text-sm text-[var(--color-text-muted)]">Connected</p>
                </div>
              </div>
              <Button variant="danger" size="sm" onClick={() => setConfirmingDeleteId(connection.id)}>
                <Trash2 className="h-4 w-4" />
                Disconnect
              </Button>
            </div>

            <dl className="mt-4 space-y-2 text-sm">
              <div className="flex justify-between">
                <dt className="text-[var(--color-text-faint)]">Installation ID</dt>
                <dd className="font-mono text-[var(--color-text)]">{connection.installation_id}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-[var(--color-text-faint)]">Connected at</dt>
                <dd className="text-[var(--color-text-muted)]">
                  {new Date(connection.created_at).toLocaleString()}
                </dd>
              </div>
              {reposData && (
                <div className="flex justify-between">
                  <dt className="text-[var(--color-text-faint)]">Repositories</dt>
                  <dd className="text-[var(--color-text-muted)]">{reposData.total_count} available</dd>
                </div>
              )}
            </dl>

            {reposData?.repositories?.length ? (
              <div className="mt-4 max-h-64 overflow-y-auto rounded-md border border-[var(--color-border)]">
                {reposData.repositories.slice(0, 20).map((r) => (
                  <a
                    key={r.id}
                    href={r.html_url}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center justify-between border-t border-[var(--color-border)] px-3 py-2 text-sm first:border-t-0 hover:bg-[var(--color-surface-2)]"
                  >
                    <span className="font-mono text-[var(--color-text)]">{r.full_name}</span>
                    <ExternalLink className="h-3.5 w-3.5 text-[var(--color-text-faint)]" />
                  </a>
                ))}
              </div>
            ) : null}

            <div className="mt-6 flex gap-2">
              <Link to="/projects/$projectId/github_services/new" params={{ projectId }}>
                <Button size="sm">New GitHub service</Button>
              </Link>
              {installUrl && (
                <a href={installUrl} target="_blank" rel="noreferrer">
                  <Button variant="outline" size="sm">
                    <Github className="h-4 w-4" />
                    Manage on GitHub
                  </Button>
                </a>
              )}
            </div>
          </section>
        </div>
      ) : (
        <div className="mt-6 space-y-6">
          {installationIdFromQuery && (
            <ErrorBanner
              className="mb-2"
              message={
                createConnection.error
                  ? getErrorMessage(createConnection.error)
                  : createConnection.isPending
                    ? "Connecting…"
                    : ""
              }
            />
          )}
          <section className="rounded-lg border border-dashed border-[var(--color-border-strong)] p-8 text-center">
            <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-[var(--color-surface-2)]">
              <Github className="h-6 w-6 text-[var(--color-text-muted)]" />
            </div>
            <h3 className="mt-4 font-display text-lg font-semibold text-[var(--color-text)]">Connect GitHub</h3>
            <p className="mx-auto mt-2 max-w-md text-sm text-[var(--color-text-muted)]">
              Install the TysonCloud GitHub App on your repositories to enable push-to-deploy. After installing, you will be
              redirected back here to complete the connection.
            </p>
            <div className="mt-6 flex flex-col items-center justify-center gap-3 sm:flex-row">
              {installUrlWithState ? (
                <Button onClick={handleInstall}>
                  <Github className="h-4 w-4" />
                  Integrate with GitHub
                  <ExternalLink className="h-4 w-4" />
                </Button>
              ) : appLoading ? (
                <p className="text-sm text-[var(--color-text-faint)]">Loading GitHub App…</p>
              ) : (
                <p className="text-sm text-[var(--color-text-faint)]">
                  GitHub App not configured. Ask an admin to set <code className="font-mono">GITHUB_APP_SLUG</code> on the
                  backend.
                </p>
              )}
            </div>
            {installUrlWithState && (
              <p className="mt-3 text-xs text-[var(--color-text-faint)]">
                A popup will open to GitHub to authorize and select repositories — you’ll return here automatically.
              </p>
            )}

            <div className="mx-auto mt-8 max-w-md rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-4 text-left">
              <Label htmlFor="installation_id">Or enter installation ID manually</Label>
              <p className="mt-1 text-xs text-[var(--color-text-muted)]">
                After installing, GitHub appends <code className="font-mono">installation_id</code> to the redirect URL. Paste it
                here if auto-detection didn’t work.
              </p>
              <form
                className="mt-3 flex gap-2"
                onSubmit={(e) => {
                  e.preventDefault();
                  if (!installationIdInput.trim()) return;
                  createConnection.mutate({ installation_id: installationIdInput.trim() });
                }}
              >
                <Input
                  id="installation_id"
                  value={installationIdInput}
                  onChange={(e) => setInstallationIdInput(e.target.value)}
                  placeholder="12345678"
                  className="font-mono"
                />
                <Button type="submit" disabled={createConnection.isPending}>
                  {createConnection.isPending ? "Connecting…" : "Connect"}
                </Button>
              </form>
              {createConnection.error && (
                <ErrorBanner className="mt-3" message={getErrorMessage(createConnection.error)} />
              )}
            </div>
          </section>
        </div>
      )}

      <DeleteConfirmDialog
        open={!!confirmingDeleteId}
        onOpenChange={(open) => !open && setConfirmingDeleteId(null)}
        resourceName={confirmingConnection ? `installation ${confirmingConnection.installation_id}` : ""}
        resourceLabel="GitHub connection"
        pending={deleteConnection.isPending}
        error={deleteConnection.error ? getErrorMessage(deleteConnection.error) : undefined}
        onConfirm={() => {
          if (!confirmingDeleteId) return;
          deleteConnection.mutate(confirmingDeleteId, {
            onSuccess: () => setConfirmingDeleteId(null),
          });
        }}
      />
    </main>
  );
}
