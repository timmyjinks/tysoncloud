import { useEffect } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useAuth } from "@clerk/clerk-react";
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
  const { isLoaded, isSignedIn } = useAuth();

  const hasOpener = () => {
    try {
      return !!window.opener && !window.opener.closed;
    } catch {
      return false;
    }
  };

  const isPopup = () => {
    try {
      return hasOpener() || window.name === "tysoncloud-github-install";
    } catch {
      return hasOpener();
    }
  };

  const forceClose = () => {
    try {
      window.close();
    } catch {}
    try {
      window.self.close();
    } catch {}
    try {
      window.top?.close();
    } catch {}
  };

  const notifyOpener = (installationId: string) => {
    try {
      if (hasOpener()) {
        window.opener.postMessage(
          { type: "github-app-installed", installation_id: installationId, state },
          "*",
        );
        return true;
      }
    } catch {}
    return false;
  };

  useEffect(() => {
    if (!installation_id) return;
    if (hasOpener()) {
      notifyOpener(installation_id);
      forceClose();
    }
  }, [installation_id, state]);

  useEffect(() => {
    if (hasOpener()) return;
    if (!isLoaded || !isSignedIn || !installation_id) return;
    if (createConnection.isSuccess || createConnection.isPending || createConnection.isError) return;
    createConnection.mutate(
      { installation_id: Number(installation_id) },
      {
        onSuccess: () => {
          if (isPopup()) {
            forceClose();
            setTimeout(() => forceClose(), 200);
            return;
          }
          const projectId = state;
          if (projectId) {
            navigate({
              to: "/projects/$projectId",
              params: { projectId },
              search: {} as never,
            });
          } else {
            navigate({ to: "/dashboard" });
          }
        },
      },
    );
  }, [isLoaded, isSignedIn, installation_id, state, createConnection, navigate]);

  if (!installation_id) {
    return (
      <main className="mx-auto max-w-xl px-4 py-16 text-center">
        <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">GitHub callback</h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">
          No installation ID was provided. Try installing the app again from the project page.
        </p>
        <Button className="mt-6" onClick={() => navigate({ to: "/dashboard" })}>
          Back to dashboard
        </Button>
      </main>
    );
  }

  if (hasOpener()) {
    return (
      <main className="mx-auto max-w-xl px-4 py-16 text-center">
        <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">Connecting GitHub…</h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">Installation ID: {installation_id}</p>
        <p className="mt-4 text-sm text-[var(--color-text-faint)]">Finishing in parent window…</p>
        <p className="mt-2 text-sm text-[var(--color-text-faint)]">If this window does not close, click below.</p>
        <Button
          className="mt-6"
          onClick={() => {
            notifyOpener(installation_id!);
            forceClose();
          }}
        >
          Close window
        </Button>
      </main>
    );
  }

  if (isPopup()) {
    if (!isLoaded) {
      return (
        <main className="mx-auto max-w-xl px-4 py-16 text-center">
          <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">Connecting GitHub…</h1>
          <p className="mt-2 text-sm text-[var(--color-text-muted)]">Installation ID: {installation_id}</p>
          <p className="mt-4 text-sm text-[var(--color-text-faint)]">Loading…</p>
          <Button className="mt-6" onClick={() => forceClose()}>
            Close window
          </Button>
        </main>
      );
    }
    if (!isSignedIn) {
      return (
        <main className="mx-auto max-w-xl px-4 py-16 text-center">
          <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">Sign in required</h1>
          <p className="mt-2 text-sm text-[var(--color-text-muted)]">Please sign in to complete the GitHub connection.</p>
          <Button className="mt-6" onClick={() => navigate({ to: "/sign-in" })}>
            Sign in
          </Button>
        </main>
      );
    }
    return (
      <main className="mx-auto max-w-xl px-4 py-16 text-center">
        <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">Connecting GitHub…</h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">Installation ID: {installation_id}</p>
        {createConnection.isPending && <p className="mt-4 text-sm text-[var(--color-text-faint)]">Saving connection…</p>}
        {createConnection.isSuccess && <p className="mt-4 text-sm text-[var(--color-text-faint)]">Connected. Closing…</p>}
        {createConnection.isSuccess && (
          <Button className="mt-6" onClick={() => forceClose()}>
            Close window
          </Button>
        )}
        {createConnection.isError && (
          <ErrorBanner className="mt-6" message={getErrorMessage(createConnection.error)} />
        )}
        {createConnection.isError && (
          <div className="mt-4 flex justify-center gap-2">
            <Button onClick={() => createConnection.mutate({ installation_id: Number(installation_id) })}>Retry</Button>
            <Button variant="outline" onClick={() => forceClose()}>
              Close window
            </Button>
          </div>
        )}
      </main>
    );
  }

  if (!isLoaded) {
    return (
      <main className="mx-auto max-w-xl px-4 py-16 text-center">
        <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">Connecting GitHub…</h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">Installation ID: {installation_id}</p>
        <p className="mt-4 text-sm text-[var(--color-text-faint)]">Loading…</p>
      </main>
    );
  }

  if (!isSignedIn) {
    return (
      <main className="mx-auto max-w-xl px-4 py-16 text-center">
        <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">Sign in required</h1>
        <p className="mt-2 text-sm text-[var(--color-text-muted)]">Please sign in to complete the GitHub connection.</p>
        <Button className="mt-6" onClick={() => navigate({ to: "/sign-in" })}>
          Sign in
        </Button>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-xl px-4 py-16 text-center">
      <h1 className="font-display text-2xl font-semibold text-[var(--color-text)]">Connecting GitHub…</h1>
      <p className="mt-2 text-sm text-[var(--color-text-muted)]">Installation ID: {installation_id}</p>
      {createConnection.isPending && <p className="mt-4 text-sm text-[var(--color-text-faint)]">Saving connection…</p>}
      {createConnection.isSuccess && <p className="mt-4 text-sm text-[var(--color-text-faint)]">Connected.</p>}
      {createConnection.isError && (
        <ErrorBanner className="mt-6" message={getErrorMessage(createConnection.error)} />
      )}
      {createConnection.isError && (
        <Button className="mt-4" onClick={() => createConnection.mutate({ installation_id: Number(installation_id) })}>
          Retry
        </Button>
      )}
    </main>
  );
}
