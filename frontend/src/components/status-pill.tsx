import { cn } from "@/lib/utils";

/**
 * Only services carry a status today (see backend/store/services.go) — databases have
 * no status field, so this pill is service-only. Anything not recognized falls back to
 * the neutral "in progress" treatment rather than guessing at good/bad.
 */
function classify(status: string): { label: string; tone: "good" | "bad" | "warn" } {
  const normalized = status.toLowerCase();
  if (normalized === "running") return { label: "Live", tone: "good" };
  if (normalized.includes("fail") || normalized.includes("error")) {
    return { label: "Failed", tone: "bad" };
  }
  return { label: status || "Deploying", tone: "warn" };
}

export function StatusPill({ status, className }: { status: string; className?: string }) {
  const { label, tone } = classify(status);
  return (
    <span
      className={cn(
        "inline-block rounded-full px-3 py-1 text-center text-xs font-medium capitalize",
        tone === "good" && "bg-[var(--color-good-soft)] text-[var(--color-good)]",
        tone === "bad" && "bg-[var(--color-bad-soft)] text-[var(--color-bad)]",
        tone === "warn" && "bg-[var(--color-warn-soft)] text-[var(--color-warn)]",
        className,
      )}
    >
      {label}
    </span>
  );
}
