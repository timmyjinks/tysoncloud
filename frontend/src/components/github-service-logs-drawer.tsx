import { useGithubLogStream } from "@/lib/logs/use-log-stream";
import { LogsDrawer } from "@/components/logs-drawer";

type GithubServiceLogsDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
  githubServiceId: string;
  serviceName: string;
};

export function GithubServiceLogsDrawer({ open, onOpenChange, projectId, githubServiceId, serviceName }: GithubServiceLogsDrawerProps) {
  const { lines, status, clear, firstLineNumber } = useGithubLogStream(projectId, githubServiceId, open);
  return (
    <LogsDrawer
      open={open}
      onOpenChange={onOpenChange}
      serviceName={serviceName}
      lines={lines}
      status={status}
      clear={clear}
      firstLineNumber={firstLineNumber}
    />
  );
}
