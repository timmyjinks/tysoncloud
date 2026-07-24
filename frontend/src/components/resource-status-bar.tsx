type ResourceStatusBarProps = {
  serviceCount: number;
  databaseCount: number;
  runningCount: number;
  projectId: string;
};

export function ResourceStatusBar({
  serviceCount,
  databaseCount,
  runningCount,
  projectId,
}: ResourceStatusBarProps) {
  const notRunning = serviceCount - runningCount;

  return (
    <div className="mt-3 flex items-center justify-between rounded-md bg-[var(--color-surface-2)] px-4 py-2 font-mono text-xs text-[var(--color-text-faint)]">
      <span className="flex items-center gap-4">
        <span>
          <span className="text-[var(--color-good)]">●</span> {runningCount} running
        </span>
        {notRunning > 0 && (
          <span>
            <span className="text-[var(--color-warn)]">●</span> {notRunning} deploying
          </span>
        )}
        <span>{databaseCount} database{databaseCount === 1 ? "" : "s"}</span>
      </span>
      <span className="truncate">{projectId}</span>
    </div>
  );
}
