import { useEffect, useRef } from "react";
import { Link } from "@tanstack/react-router";
import { UserButton } from "@clerk/clerk-react";
import { Github, LayoutGrid, Plus } from "lucide-react";
import { useProjects } from "@/lib/api/projects";
import { useCreateGithubConnection, useGithubApp, useGithubConnections } from "@/lib/api/github";
import { cn } from "@/lib/utils";

type AppSidebarProps = {
  /** projectId of the project currently open, if any — highlights it in the list */
  activeProjectId?: string;
};

export function AppSidebar({ activeProjectId }: AppSidebarProps) {
  const { data: projects } = useProjects();
  const { data: appInfo } = useGithubApp();
  const { data: connections, refetch: refetchGithubConnection } = useGithubConnections();
  const createGithubConnection = useCreateGithubConnection();
  const isGithubConnected = (connections?.length ?? 0) > 0;
  const installUrl = appInfo?.install_url || "";
  // Use a generic state so the callback can close the popup; projectId not required for global sidebar
  const installUrlWithState = installUrl
    ? `${installUrl}${installUrl.includes("?") ? "&" : "?"}state=${encodeURIComponent(activeProjectId ?? "dashboard")}`
    : "";
  const popupRef = useRef<Window | null>(null);
  const handledRef = useRef(false);

  const handleIntegrateWithGithub = () => {
    if (!installUrlWithState) return;
    handledRef.current = false;
    const w = 600;
    const h = 700;
    const left = window.screenX + (window.outerWidth - w) / 2;
    const top = window.screenY + (window.outerHeight - h) / 2;
    const popup = window.open(
      installUrlWithState,
      "tysoncloud-github-install",
      `popup=yes,width=${w},height=${h},left=${left},top=${top},scrollbars=yes`,
    );
    if (!popup) {
      window.location.href = installUrlWithState;
      return;
    }
    popupRef.current = popup;
    const timer = window.setInterval(() => {
      if (popup.closed) {
        window.clearInterval(timer);
        popupRef.current = null;
        refetchGithubConnection();
        return;
      }
      if (handledRef.current) return;
      try {
        const href = popup.location.href;
        if (href.includes("/github/callback") && href.includes("installation_id=")) {
          const url = new URL(href);
          const iid = url.searchParams.get("installation_id");
          if (iid && !handledRef.current) {
            handledRef.current = true;
            window.clearInterval(timer);
            createGithubConnection.mutate(
              { installation_id: Number(iid) },
              {
                onSuccess: () => {
                  try {
                    popup.close();
                  } catch {}
                  popupRef.current = null;
                  refetchGithubConnection();
                },
                onError: () => {
                  try {
                    popup.close();
                  } catch {}
                  popupRef.current = null;
                  refetchGithubConnection();
                },
              },
            );
          }
        }
      } catch {}
    }, 500);
  };

  useEffect(() => {
    const handler = (event: MessageEvent) => {
      if (event.data?.type !== "github-app-installed") return;
      if (handledRef.current) return;
      const installationId = event.data.installation_id as string | undefined;
      if (installationId) {
        handledRef.current = true;
        createGithubConnection.mutate(
          { installation_id: Number(installationId) },
          {
            onSuccess: () => {
              try {
                popupRef.current?.close();
              } catch {}
              popupRef.current = null;
              refetchGithubConnection();
            },
            onError: () => {
              try {
                popupRef.current?.close();
              } catch {}
              popupRef.current = null;
              refetchGithubConnection();
            },
          },
        );
      } else {
        try {
          popupRef.current?.close();
        } catch {}
        popupRef.current = null;
        refetchGithubConnection();
      }
    };
    window.addEventListener("message", handler);
    return () => window.removeEventListener("message", handler);
  }, [createGithubConnection, refetchGithubConnection]);

  return (
    <aside className="sticky top-0 flex h-screen w-72 shrink-0 flex-col border-r border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="border-b border-[var(--color-border)] px-5 py-5">
        <Link to="/dashboard" className="flex items-center gap-2.5 font-display text-lg font-semibold tracking-tight text-[var(--color-text)]">
          <span className="h-2.5 w-2.5 shrink-0 bg-[var(--color-accent)]" aria-hidden="true" />
          TYSONCLOUD
        </Link>
      </div>

      <div className="px-4 pt-4">
        <Link
          to="/dashboard"
          aria-current={!activeProjectId ? "page" : undefined}
          className={cn(
            "flex cursor-pointer items-center gap-2.5 rounded-md px-3 py-2 text-base transition-colors",
            !activeProjectId
              ? "bg-[var(--color-surface-2)] text-[var(--color-text)]"
              : "text-[var(--color-text-muted)] hover:bg-[var(--color-surface-2)] hover:text-[var(--color-text)]",
          )}
        >
          <LayoutGrid className="h-4 w-4" />
          All projects
        </Link>
      </div>

      <div className="mt-6 flex items-center justify-between px-5">
        <span className="font-mono text-xs font-medium tracking-wider text-[var(--color-text-faint)] uppercase">
          Projects
        </span>
        <Link
          to="/dashboard/new"
          aria-label="New project"
          className="text-[var(--color-text-faint)] transition-colors hover:text-[var(--color-accent)]"
        >
          <Plus className="h-4 w-4" />
        </Link>
      </div>

      <nav className="mt-3 flex-1 space-y-0.5 overflow-y-auto px-4 pb-4">
        {projects?.map((project) => (
          <Link
            key={project.id}
            to="/projects/$projectId"
            params={{ projectId: project.id }}
            aria-current={activeProjectId === project.id ? "page" : undefined}
            className={cn(
              "flex cursor-pointer items-center gap-2 truncate rounded-md px-3 py-2 font-mono text-base transition-colors",
              activeProjectId === project.id
                ? "bg-[var(--color-accent-soft)] text-[var(--color-accent)]"
                : "text-[var(--color-text-muted)] hover:bg-[var(--color-surface-2)] hover:text-[var(--color-text)]",
            )}
          >
            <span className="truncate">{project.name}</span>
          </Link>
        ))}
        {projects && projects.length === 0 && (
          <p className="px-3 py-2 text-sm text-[var(--color-text-faint)]">No projects yet.</p>
        )}
      </nav>

      <div className="border-t border-[var(--color-border)] px-4 py-4">
        <div className="mb-2 flex items-center justify-between px-1">
          <span className="font-mono text-xs font-medium tracking-wider text-[var(--color-text-faint)] uppercase">
            Integrations
          </span>
        </div>
        <button
          type="button"
          onClick={handleIntegrateWithGithub}
          title={isGithubConnected ? "GitHub is connected — click to manage/reinstall" : "Connect GitHub"}
          className="flex w-full items-center gap-2.5 rounded-md border border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-2 text-sm transition-colors hover:bg-[var(--color-surface-hover)]"
        >
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-[#24292f] text-white">
            <Github className="h-4 w-4" />
          </span>
          <span className="flex-1 text-left font-medium text-[var(--color-text)]">GitHub</span>
          <span className="flex items-center gap-1.5">
            <span
              className={`inline-flex h-2 w-2 rounded-full ${isGithubConnected ? "bg-emerald-500" : "bg-zinc-400"}`}
              aria-hidden
            />
            <span className={`text-xs ${isGithubConnected ? "text-emerald-600" : "text-[var(--color-text-faint)]"}`}>
              {isGithubConnected ? "Connected" : "Not connected"}
            </span>
          </span>
        </button>
      </div>

      <div className="flex items-center justify-between border-t border-[var(--color-border)] px-5 py-4">
        <UserButton />
        <span className="font-mono text-xs text-[var(--color-text-faint)]">v0.1</span>
      </div>
    </aside>
  );
}
