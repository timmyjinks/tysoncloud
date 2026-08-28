import { useEffect } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useCreateGithubConnection } from "@/lib/api/github";
import { getErrorMessage } from "@/lib/api/client";
import { ErrorBanner } from "@/components/error-banner";
import { Button } from "@/components/ui/button";

export const Route = createFileRoute("/github/callback")({
  validateSearch: (search: Record<string, unknown>) => ({
    installation_id: typeof search.installation_id === "string" ? search.installation_id : undefined,
    setup_action: typeof search.setup_action === "string" ? search.setup_action : undefined,
    state: typeof search.state === "string" ? search.state : undefined,
  }),
  component: GithubCallbackPage,
});

function GithubCallbackPage() {
  const navigate = useNavigate();
  const { installation_id, state } = Route.useSearch();
  const createConnection = useCreateGithubConnection();

  // Notify opener if we're inside the install popup, then close ourselves
  const notifyOpenerAndClose = (installationId: string) => {
    try {
      if (window.opener && !window.opener.closed) {
        window.opener.postMessage(
          { type: "github-app-installed", installation_id: installationId, state },
          window.location.origin,
        );
        window.close();
        return true;
      }
    } catch {
      // ignore cross-origin / already-closed errors
    }
    return false;
  };

  useEffect(() => {
    if (installation_id && !createConnection.isSuccess && !createConnection.isPending && !createConnection.isError) {
      createConnection.mutate(
        { installation_id },
        {
          onSuccess: () => {
            if (notifyOpenerAndClose(installation_id)) return;
            const projectId = state;
            if (projectId) {
              navigate({
                to: "/projects/$projectId/integrations",
                params: { projectId },
                search: {} as never,
              });
            } else {
              navigate({ to: "/dashboard" });
            }
          },
        },
      );
    }
  }, [installation_id, state, createConnection, navigate]);

  // If mutation already succeeded on mount (e.g. retry), still try to close popup
  useEffect(() => {
    if (createConnection.isSuccess && installation_id) {
      notifyOpenerAndClose(installation_id);
    }
  }, [createConnection.isSuccess, installation_id, state]);

  if (!installation_id) {
    return (
      <main className="mx-auto max-w-xl px-4 py-16 text-center">
        <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">GitHub callback</h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">
          No installation ID was provided. Try installing the app again from Integrations.
        </p>
        <Button className="mt-6" onClick={() => navigate({ to: "/dashboard" })}>
          Back to dashboard
        </Button>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-xl px-4 py-16 text-center">
      <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">Connecting GitHub…</h1>
      <p className="mt-2 text-sm text-[var(--color-text-muted)]">Installation ID: {installation_id}</p>
      {createConnection.isPending && <p className="mt-4 text-sm text-[var(--color-text-faint)]">Saving connection…</p>}
      {createConnection.isError && (
        <ErrorBanner className="mt-6" message={getErrorMessage(createConnection.error)} />
      )}
      {createConnection.isError && (
        <Button className="mt-4" onClick={() => createConnection.mutate({ installation_id })}>
          Retry
        </Button>
      )}
    </main>
  );
}
