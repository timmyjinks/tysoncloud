import { useMemo, useState } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Github, Lock, Search, X } from "lucide-react";
import { useCreateGithubService, useGithubConnections, useGithubRepos } from "@/lib/api/github";
import { getErrorMessage } from "@/lib/api/client";
import { FormShell } from "@/components/form-shell";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ErrorBanner } from "@/components/error-banner";
import { SERVICE_RESOURCE_LIMITS } from "@/lib/resource-limits";

export const Route = createFileRoute("/projects/$projectId/github_services/new")({
  component: NewGithubServicePage,
});

function NewGithubServicePage() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate();
  const createGithubService = useCreateGithubService(projectId);
  const { data: connections } = useGithubConnections();
  const installationId = connections?.[0]?.installation_id ? String(connections[0].installation_id) : "";
  const { data: reposData, isLoading: reposLoading, error: reposError } = useGithubRepos(installationId);

  const [name, setName] = useState("");
  const [repo, setRepo] = useState("");
  const [repoId, setRepoId] = useState("");
  const [port, setPort] = useState("3000");
  const [rootDir, setRootDir] = useState("");
  const [domain, setDomain] = useState("");
  const [env, setEnv] = useState("");
  const [filter, setFilter] = useState("");
  const [repoError, setRepoError] = useState<string | null>(null);

  const repos = useMemo(() => reposData?.repositories ?? [], [reposData]);

  const filteredRepos = useMemo(() => {
    if (!filter) return repos;
    const q = filter.toLowerCase();
    return repos.filter((r) => r.full_name.toLowerCase().includes(q));
  }, [repos, filter]);

  const hasConnection = !!installationId;

  return (
    <FormShell
      backTo="/projects/$projectId"
      backLabel="Back to project"
      title="New GitHub service"
      description="Build and deploy a container from a GitHub repository."
      onSubmit={(e) => {
        e.preventDefault();
        if (!repo || !repoId) {
          setRepoError("Please select a repository from the list.");
          return;
        }
        setRepoError(null);
        const trimmed = domain.trim();
        const toPayload = (raw: string): string | undefined => {
          let v = raw.trim();
          if (!v) return undefined;
          if (v.startsWith("tc-")) v = v.slice(3);
          if (v.includes(".")) v = v.split(".")[0]!;
          return v;
        };
        const payloadDomain = toPayload(trimmed);
        createGithubService.mutate(
          {
            name,
            repo,
            repo_id: Number(repoId),
            port: Number(port),
            domain: payloadDomain,
            root_dir: rootDir.trim() || ".",
            env,
          },
          { onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }) },
        );
      }}
      error={createGithubService.error ? getErrorMessage(createGithubService.error) : undefined}
      pending={createGithubService.isPending}
      submitLabel="Deploy from GitHub"
      pendingLabel="Deploying…"
      cancelTo="/projects/$projectId"
    >
      {!hasConnection && <ErrorBanner message="No GitHub connection found. Connect GitHub from the sidebar Integrations section." />}

      <div>
        <Label htmlFor="name">Service name</Label>
        <Input
          id="name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="web"
          className="mt-2"
        />
      </div>

      <div>
        <Label htmlFor="repo-filter">Repository</Label>
        {reposLoading ? (
          <p className="mt-2 text-sm text-[var(--color-text-faint)]">loading repositories…</p>
        ) : reposError ? (
          <ErrorBanner className="mt-2" message={getErrorMessage(reposError)} />
        ) : repos.length === 0 ? (
          <p className="mt-2 text-sm text-[var(--color-text-faint)]">
            No repositories found. Install the GitHub App on a repo and refresh.
          </p>
        ) : (
          <>
            <div className="relative mt-2">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--color-text-faint)]" />
              <Input
                id="repo-filter"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="Search repositories…"
                className="pl-9"
              />
            </div>

            {repo && (
              <div className="mt-3 flex items-center justify-between rounded-md border border-[var(--color-accent)] bg-[var(--color-accent-soft)] px-3 py-2">
                <span className="flex items-center gap-2 text-sm">
                  <Github className="h-4 w-4 shrink-0 text-[var(--color-accent)]" />
                  <span className="font-mono font-medium text-[var(--color-accent)]">{repo}</span>
                </span>
                <button
                  type="button"
                  onClick={() => {
                    setRepo("");
                    setRepoId("");
                  }}
                  className="cursor-pointer rounded p-1 text-[var(--color-accent)] hover:bg-black/5"
                  aria-label="Clear selected repository"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            )}

            <div className="mt-2 overflow-hidden rounded-md border border-[var(--color-border)] bg-[var(--color-surface)]">
              <div className="max-h-64 overflow-y-auto divide-y divide-[var(--color-border)]">
                {filteredRepos.length === 0 ? (
                  <p className="px-3 py-8 text-center text-sm text-[var(--color-text-faint)]">No matches.</p>
                ) : (
                  filteredRepos.map((r) => {
                    const selected = repo === r.full_name;
                    return (
                      <button
                        key={r.id}
                        type="button"
                        onClick={() => {
                          setRepo(r.full_name);
                          setRepoId(String(r.id));
                          setRepoError(null);
                        }}
                        className={`flex w-full items-center justify-between px-3 py-2 text-left text-sm transition-colors hover:bg-[var(--color-surface-2)] ${selected ? "bg-[var(--color-accent-soft)] text-[var(--color-accent)]" : "text-[var(--color-text)]"}`}
                      >
                        <span className="truncate font-mono">{r.full_name}</span>
                        <span className="ml-2 inline-flex shrink-0 items-center gap-1 text-xs text-[var(--color-text-faint)]">
                          {r.private && <Lock className="h-3 w-3" />}
                          {r.private ? "private" : "public"}
                        </span>
                      </button>
                    );
                  })
                )}
              </div>
              <div className="border-t border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-2 text-xs text-[var(--color-text-faint)]">
                {filteredRepos.length} of {repos.length} repositories
                {repo ? " · selected" : ""}
              </div>
            </div>
          </>
        )}
        {repoError && <p className="mt-2 text-xs text-red-500">{repoError}</p>}
        {!repo && !reposLoading && !reposError && repos.length > 0 && (
          <p className="mt-2 text-xs text-[var(--color-text-muted)]">Select a repository from the list above.</p>
        )}
      </div>

      <div>
        <Label htmlFor="root_dir">Root directory</Label>
        <Input
          id="root_dir"
          value={rootDir}
          onChange={(e) => setRootDir(e.target.value)}
          placeholder="/ (repository root)"
          className="mt-2 font-mono"
        />
        <p className="mt-1 text-xs text-[var(--color-text-muted)]">
          Leave empty for the repository root, or enter a subdirectory (e.g.{" "}
          <code className="font-mono">apps/web</code>).
        </p>
      </div>

      <div>
        <Label htmlFor="port">Port</Label>
        <Input
          id="port"
          type="number"
          required
          value={port}
          onChange={(e) => setPort(e.target.value)}
          className="mt-2 font-mono"
        />
      </div>

      <div>
        <Label htmlFor="domain">Custom domain</Label>
        {domain.trim().includes(".") ? (
          <Input
            id="domain"
            value={domain}
            onChange={(e) => setDomain(e.target.value)}
            placeholder="my-app.example.com"
            className="mt-2 font-mono"
            aria-describedby="domain-help domain-preview"
          />
        ) : (
          <div className="mt-2 flex items-center">
            <span className="rounded-l-md border border-r-0 border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm font-mono text-[var(--color-text-muted)]">
              tc-
            </span>
            <Input
              id="domain"
              value={domain}
              onChange={(e) => setDomain(e.target.value)}
              placeholder="my-app"
              className="rounded-none font-mono"
              aria-describedby="domain-help domain-preview"
            />
            <span className="rounded-r-md border border-l-0 border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 text-sm font-mono text-[var(--color-text-muted)]">
              .tysonjenkins.dev
            </span>
          </div>
        )}
        <p id="domain-help" className="mt-1 text-xs text-[var(--color-text-muted)]">
          Optional. Leave blank for an auto-generated domain.
        </p>
        <p id="domain-preview" className="mt-1 text-xs text-[var(--color-text-faint)]">
          Preview:{" "}
          <code className="font-mono">
            https://
            {(() => {
              let v = domain.trim();
              if (!v) return "tc-my-app.tysonjenkins.dev";
              if (v.startsWith("tc-")) v = v.slice(3);
              if (v.includes(".")) v = v.split(".")[0]!;
              return `tc-${v}.tysonjenkins.dev`;
            })()}
          </code>
        </p>
      </div>

      <div>
        <Label htmlFor="env">Environment variables</Label>
        <Textarea
          id="env"
          value={env}
          onChange={(e) => setEnv(e.target.value)}
          placeholder={"KEY=value\nANOTHER_KEY=value"}
          rows={5}
          className="mt-2"
        />
        <p className="mt-1 text-xs text-[var(--color-text-muted)]">
          One <code>KEY=value</code> pair per line. Optional.
        </p>
      </div>

      <p className="text-xs text-[var(--color-text-muted)]">
        Each service runs with a maximum of <code className="font-mono">{SERVICE_RESOURCE_LIMITS.cpu}</code> and{" "}
        <code className="font-mono">{SERVICE_RESOURCE_LIMITS.memory}</code> memory. Pushes to the repo will auto-redeploy.
      </p>


    </FormShell>
  );
}
