import { useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { FileCode } from "lucide-react";
import { useApplyProjectConfig } from "@/lib/api/project-config";
import { serviceKeys } from "@/lib/api/services";
import { databaseKeys } from "@/lib/api/databases";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { ApiRequestError } from "@/lib/api/client";

export const Route = createFileRoute("/projects/$projectId/config")({
  component: ProjectConfigPage,
});

const EXAMPLE_TOML = `# Define the resources this project should have, then apply.
# This runs once — it creates what's listed below, it doesn't sync or delete.

[[services]]
name = "web"
image = "myorg/web:latest"
port = 3000

  [services.volume]
  mount_path = "/data"
  storage_gb = 10

[[services]]
name = "worker"
image = "myorg/worker:latest"
port = 8080

[[databases]]
name = "primary"
engine = "postgres"
storage_gb = 20
`;

function ProjectConfigPage() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const applyConfig = useApplyProjectConfig(projectId);

  const [content, setContent] = useState(EXAMPLE_TOML);

  const issues =
    applyConfig.error instanceof ApiRequestError ? applyConfig.error.body?.issues : undefined;

  return (
    <main className="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to project
      </Link>

      <div className="mt-4 mb-2 flex items-center gap-2">
        <FileCode className="h-5 w-5 text-[var(--color-text-faint)]" />
        <h1 className="font-mono text-2xl font-bold">Apply config</h1>
      </div>
      <p className="mb-6 text-sm text-[var(--color-text-muted)]">
        Define services, databases, and volumes as TOML and apply them in one shot. This creates
        resources — it's a one-time action, not a saved file.
      </p>

      <form
        className="space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          applyConfig.mutate(
            { content },
            {
              onSuccess: () => {
                qc.invalidateQueries({ queryKey: serviceKeys.byProject(projectId) });
                qc.invalidateQueries({ queryKey: databaseKeys.byProject(projectId) });
                navigate({ to: "/projects/$projectId", params: { projectId } });
              },
            },
          );
        }}
      >
        <Textarea
          id="config"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          spellCheck={false}
          rows={24}
          className="text-sm leading-relaxed"
        />

        {applyConfig.error && (
          <div className="rounded-md border border-[var(--color-bad)] bg-[var(--color-bad-soft)] p-3">
            <p className="text-sm text-[var(--color-bad)]">{applyConfig.error.message}</p>
            {issues && issues.length > 0 && (
              <ul className="mt-2 space-y-1 text-xs text-[var(--color-bad)]">
                {issues.map((issue, i) => (
                  <li key={i}>
                    {issue.line != null && <span className="font-mono">line {issue.line}: </span>}
                    {issue.message}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}

        <div className="flex items-center gap-3 border-t border-[var(--color-border)] pt-5">
          <Button type="submit" disabled={applyConfig.isPending}>
            {applyConfig.isPending ? "Applying…" : "Apply config"}
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
