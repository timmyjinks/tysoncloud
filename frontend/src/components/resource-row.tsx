import { Link } from "@tanstack/react-router";
import { MoreVertical, Pencil, Trash2 } from "lucide-react";
import { StatusPill } from "@/components/status-pill";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { DropdownMenu, DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

type ResourceRowProps = {
  icon: React.ReactNode;
  name: string;
  /** Only services have a status today — omit for databases. */
  status?: string;
  /** e.g. image tag or db engine version */
  runtime: string;
  /** optional second line under runtime, e.g. resource limits */
  subtitle?: string;
  /** e.g. ":3000" or "12 GB" */
  size: string;
  /** domain text — colored as a link when href is present, muted otherwise (e.g. "internal") */
  domain: string;
  domainHref?: string;
  /** private/internal domain for services */
  privateDomain?: string;
  /** Omitted for ephemeral rows (e.g. preview instances) — name renders as plain text. */
  detailHref?: string;
  /** Omitted for read-only rows — hides the actions menu. */
  onUpdate?: () => void;
  onDelete?: () => void;
  /** When provided, renders a selection checkbox at the start of the row. */
  selected?: boolean;
  onToggleSelect?: () => void;
  className?: string;
};

export function ResourceRow({
  icon,
  name,
  status,
  runtime,
  subtitle,
  size,
  domain,
  domainHref,
  privateDomain,
  detailHref,
  onUpdate,
  onDelete,
  selected,
  onToggleSelect,
  className,
}: ResourceRowProps) {
  return (
    <div
      className={cn(
        "group flex items-center gap-5 border-t border-[var(--color-border)] px-5 py-5 first:border-t-0 first:rounded-t-lg last:rounded-b-lg hover:bg-[var(--color-surface-hover)]",
        selected && "bg-[var(--color-surface-2)]",
        className,
      )}
    >
      {onToggleSelect && (
        <Checkbox
          aria-label={`Select ${name}`}
          checked={selected}
          onChange={onToggleSelect}
          className="shrink-0"
        />
      )}

      <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-[var(--color-surface-2)] text-[var(--color-text-muted)]">
        {icon}
      </span>

      {detailHref ? (
        <Link
          to={detailHref}
          className="w-44 shrink-0 truncate font-sans text-lg font-medium text-[var(--color-text)] hover:text-[var(--color-accent)]"
        >
          {name}
        </Link>
      ) : (
        <span className="w-44 shrink-0 truncate font-sans text-lg font-medium text-[var(--color-text)]">
          {name}
        </span>
      )}

      <span className="w-24 shrink-0">{status && <StatusPill status={status} />}</span>

      <span className="flex-1 min-w-0">
        <span
          title={runtime}
          className="block truncate font-mono text-base text-[var(--color-text-faint)]"
        >
          {runtime}
        </span>
        {subtitle && (
          <span
            title={subtitle}
            className="mt-0.5 block truncate text-sm text-[var(--color-text-faint)]"
          >
            {subtitle}
          </span>
        )}
      </span>

      <span className="w-16 shrink-0 text-right font-mono text-base text-[var(--color-text-faint)]">
        {size}
      </span>

      <span className="w-52 shrink-0 text-right font-mono text-base">
        <span title={domain} className="block truncate">
          {domainHref ? (
            <a
              href={domainHref}
              target="_blank"
              rel="noreferrer"
              className="text-[var(--color-accent)] hover:text-[var(--color-accent-hover)]"
            >
              {domain}
            </a>
          ) : (
            <span className="text-[var(--color-text-faint)]">{domain}</span>
          )}
        </span>
        {privateDomain && (
          <span
            title={privateDomain}
            className="mt-0.5 block truncate text-xs text-[var(--color-text-faint)]"
          >
            {privateDomain}
          </span>
        )}
      </span>

      {(onUpdate || onDelete) && (
        <DropdownMenu
          trigger={
            <Button
              variant="ghost"
              size="icon"
              aria-label={`Actions for ${name}`}
              className="h-9 w-9 shrink-0 text-[var(--color-text-faint)]"
            >
              <MoreVertical className="h-5 w-5" />
            </Button>
          }
        >
          {onUpdate && (
            <DropdownMenuItem onClick={onUpdate}>
              <Pencil className="h-4 w-4" />
              Update
            </DropdownMenuItem>
          )}
          {onUpdate && onDelete && (
            <div className="my-1 h-px bg-[var(--color-border)]" aria-hidden="true" />
          )}
          {onDelete && (
            <DropdownMenuItem destructive onClick={onDelete}>
              <Trash2 className="h-4 w-4" />
              Delete
            </DropdownMenuItem>
          )}
        </DropdownMenu>
      )}
    </div>
  );
}