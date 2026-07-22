import { useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useCreateService } from "@/lib/api/services";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

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
  const [env, setEnv] = useState("");

  return (
    <main className="mx-auto max-w-lg px-4 py-8 sm:px-6 lg:px-8">
      <Link
        to="/projects/$projectId"
        params={{ projectId }}
        className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-accent)]"
      >
        ← Back to project
      </Link>
      <h1 className="mt-4 mb-6 font-mono text-2xl font-bold">New service</h1>

      <form
        className="space-y-5 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] p-6"
        onSubmit={(e) => {
          e.preventDefault();
          createService.mutate(
            { name, image, port: Number(port), env },
            { onSuccess: () => navigate({ to: "/projects/$projectId", params: { projectId } }) },
          );
        }}
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

        {createService.error && (
          <p className="text-sm text-[var(--color-bad)]">{createService.error.message}</p>
        )}

        <div className="flex gap-3 border-t border-[var(--color-border)] pt-5">
          <Button type="submit" disabled={createService.isPending}>
            {createService.isPending ? "Deploying…" : "Deploy service"}
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
