import { createFileRoute, Link } from "@tanstack/react-router";
import { Navbar } from "@/components/navbar";

export const Route = createFileRoute("/")({
  component: LandingPage,
});

function LandingPage() {
  return (
    <div>
      <Navbar />

      <section className="mx-auto max-w-4xl px-4 py-24 text-center sm:px-6 lg:px-8 md:py-32">
        <p className="terminal-strip mb-6 inline-block rounded-full border border-[var(--color-border)] px-3 py-1">
          <span className="prompt">$</span> tysoncloud deploy --image nginx:latest
        </p>
        <h1 className="mb-6 font-mono text-4xl font-bold tracking-tight md:text-6xl">
          Infrastructure that reads like{" "}
          <span className="text-[var(--color-accent)]">code you wrote</span>
        </h1>
        <p className="mx-auto mb-10 max-w-2xl text-lg text-[var(--color-text-muted)]">
          Push a Docker image, get a running service, a domain, and a database if you
          need one. No dashboards to babysit — just infrastructure that does what you
          told it to.
        </p>
        <div className="flex justify-center gap-3">
          <Link
            to="/sign-up"
            className="rounded-md bg-[var(--color-accent)] px-5 py-2.5 text-sm font-medium text-white hover:bg-[var(--color-accent-hover)]"
          >
            Start deploying
          </Link>
          <Link
            to="/sign-in"
            className="rounded-md border border-[var(--color-border-strong)] px-5 py-2.5 text-sm font-medium text-[var(--color-text)] hover:bg-[var(--color-surface-2)]"
          >
            Sign in
          </Link>
        </div>
      </section>

      <section className="mx-auto max-w-4xl px-4 pb-24 sm:px-6 lg:px-8">
        <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] overflow-hidden">
          <div className="terminal-strip flex items-center gap-2 border-b border-[var(--color-border)] bg-[var(--color-surface-2)] px-4 py-2">
            <span className="h-2.5 w-2.5 rounded-full bg-[var(--color-bad)]" />
            <span className="h-2.5 w-2.5 rounded-full bg-[var(--color-warn)]" />
            <span className="h-2.5 w-2.5 rounded-full bg-[var(--color-good)]" />
            <span className="ml-2">deploy.log</span>
          </div>
          <div className="space-y-1 p-6 font-mono text-sm">
            <p className="text-[var(--color-text-muted)]">
              <span className="text-[var(--color-good)]">✓</span> image pulled — nginx:latest
            </p>
            <p className="text-[var(--color-text-muted)]">
              <span className="text-[var(--color-good)]">✓</span> deployment applied — svc-web-01
            </p>
            <p className="text-[var(--color-text-muted)]">
              <span className="text-[var(--color-good)]">✓</span> route attached — web-01.tysoncloud.dev
            </p>
            <p className="text-[var(--color-text)]">
              <span className="text-[var(--color-accent)]">→</span> running on :3000
            </p>
          </div>
        </div>
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
