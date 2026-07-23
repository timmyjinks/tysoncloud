import { Link } from "@tanstack/react-router";
import { Folder, Trash2 } from "lucide-react";

type ProjectRowProps = {
  name: string;
  id: string;
  href: string;
  onDelete: () => void;
};

export function ProjectRow({ name, id, href, onDelete }: ProjectRowProps) {
  return (
    <div className="group flex items-center gap-4 border-t border-[var(--color-border)] px-4 py-4 first:border-t-0 hover:bg-[var(--color-surface-hover)]">
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[var(--color-surface-2)] text-[var(--color-text-muted)]">
        <Folder className="h-4 w-4" />
      </span>

      <Link to={href} className="flex-1 truncate text-base font-medium text-[var(--color-text)]">
        {name}
      </Link>

      <span className="shrink-0 truncate font-mono text-sm text-[var(--color-text-faint)]">
        {id}
      </span>

      <button
        onClick={onDelete}
        aria-label={`Delete ${name}`}
        className="shrink-0 cursor-pointer text-[var(--color-text-faint)] opacity-0 transition-opacity hover:text-[var(--color-bad)] group-hover:opacity-100"
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </div>
  );
}
