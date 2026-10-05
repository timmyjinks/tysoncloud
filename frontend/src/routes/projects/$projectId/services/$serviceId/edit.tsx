import { useEffect, useState } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useService, useUpdateService } from "@/lib/api/services";
import { getErrorMessage } from "@/lib/api/client";
import { formatEnvLines } from "@/lib/utils";
import { ErrorBanner } from "@/components/error-banner";
import { FormShell } from "@/components/form-shell";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

export const Route = createFileRoute("/projects/$projectId/services/$serviceId/edit")({
  component: EditServicePage,
});

function EditServicePage() {
  const { projectId, serviceId } = Route.useParams();
  const navigate = useNavigate();
  const { data: service, isLoading, error, refetch } = useService(serviceId);
  const updateService = useUpdateService(projectId, serviceId);

  const [name, setName] = useState("");
  const [image, setImage] = useState("");
  const [port, setPort] = useState("");
  const [domain, setDomain] = useState("");
  const [env, setEnv] = useState("");

  function extractDomainMiddle(publicDomain: string): string {
    let v = publicDomain.trim();
    // Only prefill if it looks like a custom domain (tc-<name>.tysonjenkins.dev)
    // Auto domains (e.g. svc-xxx...) should leave the field blank to avoid
    // treating auto as custom and causing duplicate checks on save.
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
    setImage(service.image);
    setPort(String(service.port));
    setDomain(extractDomainMiddle(service.public_domain ?? ""));
    setEnv(formatEnvLines(service.env ?? {}));
  }, [service]);

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

  if (isLoading || !service) {
    return (
      <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
        <p className="text-sm text-[var(--color-text-faint)]">loading service…</p>
      </main>
    );
  }

  return (
    <FormShell
      backTo="/projects/$projectId/services/$serviceId"
      backLabel="Back to service"
      title="Update service"
      onSubmit={(e) => {
        e.preventDefault();
        const payloadDomain = toPayloadDomain(domain);
        updateService.mutate(
          { name, image, port: Number(port), domain: payloadDomain, env },
          {
            onSuccess: () =>
              navigate({
                to: "/projects/$projectId/services/$serviceId",
                params: { projectId, serviceId },
                search: { env: undefined },
              }),
          },
        );
      }}
      error={updateService.error ? getErrorMessage(updateService.error) : undefined}
      pending={updateService.isPending}
      submitLabel="Save changes"
      pendingLabel="Saving…"
      cancelTo="/projects/$projectId/services/$serviceId"
    >
      <div>
        <Label htmlFor="name">Service name</Label>
        <Input
          id="name"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="mt-2"
        />
      </div>

      <div>
        <Label htmlFor="image">Docker image</Label>
        <Input
          id="image"
          required
          value={image}
          onChange={(e) => setImage(e.target.value)}
          className="mt-2 font-mono"
        />
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
          Optional. Clear to revert to auto-generated domain. Type a short name (e.g.{" "}
          <code className="font-mono">my-app</code>) for <code className="font-mono">tc-my-app.tysonjenkins.dev</code>{" "}
          or a full domain like <code className="font-mono">my-app.example.com</code>.
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
          One <code>KEY=value</code> pair per line. Saving replaces the full set of environment
          variables with what's shown here.
        </p>
      </div>
    </FormShell>
  );
}