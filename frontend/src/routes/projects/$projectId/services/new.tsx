import { useState } from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useCreateService } from "@/lib/api/services";
import { getErrorMessage } from "@/lib/api/client";
import { FormShell } from "@/components/form-shell";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { SERVICE_RESOURCE_LIMITS } from "@/lib/resource-limits";

export const Route = createFileRoute("/projects/$projectId/services/new")({
  component: NewServicePage,
});

function NewServicePage() {
  const { projectId } = Route.useParams();
  const navigate = useNavigate();
  const createService = useCreateService(projectId);

  const [name, setName] = useState("");
  const [image, setImage] = useState("");
  const [port, setPort] = useState("3000");
  const [domain, setDomain] = useState("");
  const [env, setEnv] = useState("");

  return (
    <FormShell
      backTo="/projects/$projectId"
      backLabel="Back to project"
      title="New service"
      description="Deploy a container to this project."
      onSubmit={(e) => {
        e.preventDefault();
        const trimmed = domain.trim();
        // Backend only wants the subdomain name fragment, not the full hostname.
        // e.g. tc-ohyah.tysonjenkins.dev or tc-ohyah.domain.com -> "ohyah"
        const toPayload = (raw: string): string | undefined => {
          let v = raw.trim();
          if (!v) return undefined;
          if (v.startsWith("tc-")) v = v.slice(3);
          if (v.includes(".")) v = v.split(".")[0]!;
          return v;
        };
        const payloadDomain = toPayload(trimmed);
        createService.mutate(
          { name, image, port: Number(port), domain: payloadDomain, env },
          { onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }) },
        );
      }}
      error={createService.error ? getErrorMessage(createService.error) : undefined}
      pending={createService.isPending}
      submitLabel="Deploy service"
      pendingLabel="Deploying…"
      cancelTo="/projects/$projectId"
    >
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
        <Label htmlFor="image">Docker image</Label>
        <Input
          id="image"
          required
          value={image}
          onChange={(e) => setImage(e.target.value)}
          placeholder="nginx:latest"
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
          Optional. Leave blank for an auto-generated domain. Type a short name (e.g.{" "}
          <code className="font-mono">my-app</code>) for <code className="font-mono">tc-my-app.tysonjenkins.dev</code>{" "}
          or a full domain like <code className="font-mono">my-app.example.com</code>.
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
        Each instance gets up to <code className="font-mono">{SERVICE_RESOURCE_LIMITS.cpu}</code> and{" "}
        <code className="font-mono">{SERVICE_RESOURCE_LIMITS.memory}</code> of memory.
      </p>
    </FormShell>
  );
}