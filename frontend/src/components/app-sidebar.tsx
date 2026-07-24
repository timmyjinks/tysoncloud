import { Link } from "@tanstack/react-router";
import { UserButton } from "@clerk/clerk-react";
import { LayoutGrid, Plus } from "lucide-react";
import { useProjects } from "@/lib/api/projects";
import { cn } from "@/lib/utils";

type AppSidebarProps = {
  /** projectId of the project currently open, if any — highlights it in the list */
  activeProjectId?: string;
};

export function AppSidebar({ activeProjectId }: AppSidebarProps) {
  const { data: projects } = useProjects();

  return (
    <aside className="sticky top-0 flex h-screen w-60 shrink-0 flex-col border-r border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="border-b border-[var(--color-border)] px-4 py-4">
        <Link to="/dashboard" className="font-mono text-sm font-bold tracking-tight text-[var(--color-text)]">
          TYSONCLOUD
        </Link>
      </div>

      <div className="px-3 pt-3">
        <Link
          to="/dashboard"
          className={cn(
            "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors",
            !activeProjectId
              ? "bg-[var(--color-surface-2)] text-[var(--color-text)]"
              : "text-[var(--color-text-muted)] hover:bg-[var(--color-surface-2)] hover:text-[var(--color-text)]",
          )}
        >
          <LayoutGrid className="h-3.5 w-3.5" />
          All projects
        </Link>
      </div>

      <div className="mt-5 flex items-center justify-between px-4">
        <span className="font-mono text-[11px] font-medium tracking-wider text-[var(--color-text-faint)] uppercase">
          Projects
        </span>
        <Link
          to="/dashboard/new"
          aria-label="New project"
          className="text-[var(--color-text-faint)] transition-colors hover:text-[var(--color-accent)]"
        >
          <Plus className="h-3.5 w-3.5" />
        </Link>
      </div>

      <nav className="mt-2 flex-1 space-y-0.5 overflow-y-auto px-3 pb-3">
        {projects?.map((project) => (
          <Link
            key={project.id}
            to="/projects/$projectId"
            params={{ projectId: project.id }}
            className={cn(
              "flex items-center gap-1.5 truncate rounded-md px-2.5 py-1.5 font-mono text-sm transition-colors",
              activeProjectId === project.id
                ? "bg-[var(--color-accent-soft)] text-[var(--color-accent)]"
                : "text-[var(--color-text-muted)] hover:bg-[var(--color-surface-2)] hover:text-[var(--color-text)]",
            )}
          >
            <span
              className={cn(
                activeProjectId === project.id
                  ? "text-[var(--color-accent)]"
                  : "text-[var(--color-text-faint)]",
              )}
            >
              $
            </span>
            <span className="truncate">{project.name}</span>
          </Link>
        ))}
        {projects && projects.length === 0 && (
          <p className="px-2.5 py-1.5 text-xs text-[var(--color-text-faint)]">No projects yet.</p>
        )}
      </nav>

      <div className="flex items-center justify-between border-t border-[var(--color-border)] px-4 py-3">
        <UserButton />
        <span className="font-mono text-[11px] text-[var(--color-text-faint)]">v0.1</span>
      </div>
    </aside>
  );
}
