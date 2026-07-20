import { Link } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { Card } from "@/components/ui/card";
import { StatusDot } from "@/components/status-dot";
import { cn } from "@/lib/utils";

type ResourceCardProps = {
  /** Monospace prompt line, e.g. "svc-web-prod" or "db-postgres-01" */
  promptId: string;
  title: string;
  status?: string;
  /** key/value pairs rendered as a small meta grid */
  meta: { label: string; value: string; mono?: boolean; href?: string }[];
  detailHref?: string;
  onDelete?: () => void;
};

export function ResourceCard({
  promptId,
  title,
  status,
  meta,
  detailHref,
  onDelete,
}: ResourceCardProps) {
  return (
    <Card className="group overflow-hidden hover:border-[var(--color-border-strong)]">
      <div className="terminal-strip flex items-center justify-between border-b border-[var(--color-border)] bg-[var(--color-surface-2)] px-4 py-2">
        <span>
          <span className="prompt">$</span> {promptId}
        </span>
        {onDelete && (
          <button
            onClick={onDelete}
            className="cursor-pointer text-[var(--color-text-faint)] opacity-0 transition-opacity hover:text-[var(--color-bad)] group-hover:opacity-100"
            aria-label={`Delete ${title}`}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      <div className="p-4">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-base font-semibold text-[var(--color-text)]">{title}</h3>
          {status && (
            <div className="flex items-center gap-1.5">
              <StatusDot status={status} />
              <span className="text-xs capitalize text-[var(--color-text-muted)]">{status}</span>
            </div>
          )}
        </div>

        <dl className="space-y-1.5">
          {meta.map((row) => (
            <div key={row.label} className="flex items-center justify-between text-xs">
              <dt className="text-[var(--color-text-faint)]">{row.label}</dt>
              <dd
                className={cn(
                  "truncate text-[var(--color-text-muted)]",
                  row.mono && "font-mono",
                )}
              >
                {row.href ? (
                  <a
                    href={row.href}
                    target="_blank"
                    rel="noreferrer"
                    className="hover:text-[var(--color-accent)]"
                  >
                    {row.value}
                  </a>
                ) : (
                  row.value
                )}
              </dd>
            </div>
          ))}
        </dl>

        {detailHref && (
          <Link
            to={detailHref}
            className="mt-4 inline-block text-xs font-mono text-[var(--color-accent)] hover:text-[var(--color-accent-hover)]"
          >
            view details →
          </Link>
        )}
      </div>
    </Card>
  );
}
