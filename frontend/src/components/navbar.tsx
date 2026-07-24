import { Link } from "@tanstack/react-router";
import { SignedIn, SignedOut } from "@clerk/clerk-react";

export function Navbar() {
  return (
    <header className="mx-auto max-w-6xl px-4 py-6 sm:px-6 lg:px-8">
      <nav className="flex items-center justify-between">
        <Link to="/" className="flex items-center gap-2">
          <span className="font-mono text-lg font-bold tracking-tight text-[var(--color-text)]">
            TYSONCLOUD
          </span>
        </Link>
        <div className="flex items-center gap-4">
          <SignedOut>
            <Link
              to="/sign-in"
              className="text-sm text-[var(--color-text-muted)] transition-colors hover:text-[var(--color-text)]"
            >
              Sign in
            </Link>
            <Link
              to="/sign-up"
              className="rounded-md bg-[var(--color-accent)] px-3 py-1.5 text-sm font-medium text-white hover:bg-[var(--color-accent-hover)]"
            >
              Get started
            </Link>
          </SignedOut>
          <SignedIn>
            <Link
              to="/dashboard"
              className="rounded-md bg-[var(--color-accent)] px-3 py-1 text-base font-medium text-white hover:bg-[var(--color-accent-hover)]"
            >
              Dashboard
            </Link>
          </SignedIn>
        </div>
      </nav>
    </header>
  );
}
