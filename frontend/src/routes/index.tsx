import { createFileRoute, Link } from "@tanstack/react-router";
import { Database as DatabaseIcon, Server } from "lucide-react";
import { Navbar } from "@/components/navbar";
import { StatusPill } from "@/components/status-pill";

export const Route = createFileRoute("/")({
  component: LandingPage,
});

function LandingPage() {
  return (
    <div>
      <Navbar />

      <section className="mx-auto max-w-4xl px-4 py-24 text-center sm:px-6 lg:px-8 md:py-32">
        <h1 className="mb-6 font-mono text-4xl font-bold tracking-tight md:text-6xl">
          Ship a service, get{" "}
          <span className="text-[var(--color-accent)]">infrastructure that runs itself</span>
        </h1>
        <p className="mx-auto mb-10 max-w-2xl text-xl text-[var(--color-text-muted)]">
          Push a Docker image, get a running service, a domain, and a database if you
          need one. No dashboards to babysit — just infrastructure that does what you
          told it to.
        </p>
        <div className="flex justify-center gap-3">
          <Link
            to="/sign-up"
            className="rounded-md bg-[var(--color-accent)] px-6 py-3 text-base font-medium text-white hover:bg-[var(--color-accent-hover)]"
          >
            Start deploying
          </Link>
          <Link
            to="/sign-in"
            className="rounded-md border border-[var(--color-border-strong)] px-6 py-3 text-base font-medium text-[var(--color-text)] hover:bg-[var(--color-surface-2)]"
          >
            Sign in
          </Link>
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-4 pb-24 sm:px-6 lg:px-8">
        <div className="overflow-hidden rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)]">
          <div className="flex items-center justify-between border-b border-[var(--color-border)] px-5 py-3">
            <span className="text-sm font-medium text-[var(--color-text-muted)]">
              my-app <span className="text-[var(--color-text-faint)]">· 2 resources</span>
            </span>
          </div>

          <div className="flex items-center gap-4 border-b border-[var(--color-border)] px-5 py-4">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[var(--color-surface-2)] text-[var(--color-text-muted)]">
              <Server className="h-4 w-4" />
            </span>
            <span className="w-36 shrink-0 text-base font-medium">svc-web-01</span>
            <span className="w-20 shrink-0">
              <StatusPill status="running" />
            </span>
            <span className="flex-1 truncate font-mono text-sm text-[var(--color-text-faint)]">
              nginx:latest
            </span>
            <span className="w-14 shrink-0 text-right font-mono text-sm text-[var(--color-text-faint)]">
              :3000
            </span>
            <span className="w-44 shrink-0 truncate text-right font-mono text-sm text-[var(--color-accent)]">
              web-01.tysoncloud.dev
            </span>
          </div>

          <div className="flex items-center gap-4 px-5 py-4">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[var(--color-surface-2)] text-[var(--color-text-muted)]">
              <DatabaseIcon className="h-4 w-4" />
            </span>
            <span className="w-36 shrink-0 text-base font-medium">db-postgres-01</span>
            <span className="w-20 shrink-0" />
            <span className="flex-1 truncate font-mono text-sm text-[var(--color-text-faint)]">
              postgres 16
            </span>
            <span className="w-14 shrink-0 text-right font-mono text-sm text-[var(--color-text-faint)]">
              12 GB
            </span>
            <span className="w-44 shrink-0 truncate text-right font-mono text-sm text-[var(--color-text-faint)]">
              internal
            </span>
          </div>
        </div>
        <p className="mt-3 text-center text-sm text-[var(--color-text-faint)]">
          This is really what it looks like — no separate marketing screenshots.
        </p>
      </section>

      <footer className="mx-auto max-w-6xl border-t border-[var(--color-border)] px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-between gap-4 md:flex-row">
          <span className="font-mono text-sm font-bold text-[var(--color-text)]">TYSONCLOUD</span>
          <p className="text-sm text-[var(--color-text-faint)]">
            © 2026 TYSONCLOUD, Inc. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
