import { useEffect, useState } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useGithubService, useUpdateGithubService } from "@/lib/api/github";
import { getErrorMessage } from "@/lib/api/client";
import { formatEnvLines } from "@/lib/utils";
import { ErrorBanner } from "@/components/error-banner";
import { FormShell } from "@/components/form-shell";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

export const Route = createFileRoute("/projects/$projectId/github_services/$githubServiceId/edit")({
  component: EditGithubServicePage,
});

function EditGithubServicePage() {
  const { projectId, githubServiceId } = Route.useParams();
  const navigate = useNavigate();
  const { data: service, isLoading, error, refetch } = useGithubService(githubServiceId);
  const updateGithubService = useUpdateGithubService(projectId, githubServiceId);

  const [name, setName] = useState("");
  const [port, setPort] = useState("");
  const [domain, setDomain] = useState("");
  const [env, setEnv] = useState("");

  function extractDomainMiddle(publicDomain: string): string {
    let v = publicDomain.trim();
    if (!v.startsWith("tc-")) return "";
    v = v.slice(3);
    if (v.includes(".")) v = v.split(".")[0]!;
    return v;
  }

  function toPayloadDomain(raw: string): string | null {
    let v = raw.trim();
    if (!v) return null;
    if (v.startsWith("tc-")) v = v.slice(3);
    if (v.includes(".")) v = v.split(".")[0]!;
    return v;
  }

  useEffect(() => {
    if (!service) return;
    setName(service.name);
    setPort(String(service.port));
    setDomain(extractDomainMiddle(service.public_domain ?? ""));
    setEnv(formatEnvLines(service.env ?? {}));
  }, [service]);

  if (error) {
    return (
      <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
        <ErrorBanner message={getErrorMessage(error)} onRetry={() => refetch()} retryLabel="Retry" />
      </main>
    );
  }

  if (isLoading || !service) {
    return (
      <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
        <p className="text-sm text-[var(--color-text-faint)]">loading service…</p>
      </main>
    );
  }

  return (
    <FormShell
      backTo="/projects/$projectId/github_services/$githubServiceId"
      backLabel="Back to service"
      title="Update GitHub service"
      onSubmit={(e) => {
        e.preventDefault();
        const payloadDomain = toPayloadDomain(domain);
        updateGithubService.mutate(
          { name, port: Number(port), domain: payloadDomain, env },
          {
            onSuccess: () =>
              navigate({
                to: "/projects/$projectId/github_services/$githubServiceId",
                params: { projectId, githubServiceId },
              }),
          },
        );
      }}
      error={updateGithubService.error ? getErrorMessage(updateGithubService.error) : undefined}
      pending={updateGithubService.isPending}
      submitLabel="Save changes"
      pendingLabel="Saving…"
      cancelTo="/projects/$projectId/github_services/$githubServiceId"
    >
      <div className="rounded-md border border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-2 text-sm text-[var(--color-text-muted)]">
        Repository <span className="font-mono text-[var(--color-text)]">{service.repo}</span> · root{" "}
        <span className="font-mono text-[var(--color-text)]">{service.root_dir}</span> cannot be changed after creation.
      </div>

      <div>
        <Label htmlFor="name">Service name</Label>
        <Input id="name" required value={name} onChange={(e) => setName(e.target.value)} className="mt-2" />
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
          Optional. Clear to revert to auto-generated domain.
        </p>
        <p id="domain-preview" className="mt-1 text-xs text-[var(--color-text-faint)]">
          Preview:{" "}
          <code className="font-mono">
            https://
            {(() => {
              let v = domain.trim();
              if (!v) return "(auto domain)";
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
          One <code>KEY=value</code> pair per line. Saving replaces the full set.
        </p>
      </div>
    </FormShell>
  );
}
