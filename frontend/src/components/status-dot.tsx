import { cn } from "@/lib/utils";

export function StatusDot({ status, className }: { status: string; className?: string }) {
  const running = status === "running";
  return (
    <span
      className={cn(
        "inline-block h-2 w-2 rounded-full",
        running ? "bg-[var(--color-good)]" : "bg-[var(--color-bad)]",
        className,
      )}
      aria-hidden
    />
  );
}
