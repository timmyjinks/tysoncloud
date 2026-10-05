import { useGithubLogStream, usePreviewServiceLogStream } from "@/lib/logs/use-log-stream";
import { LogsDrawer } from "@/components/logs-drawer";

type GithubServiceLogsDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
  githubServiceId: string;
  serviceName: string;
  /** When present, streams the preview copy's logs instead of prod logs. */
  preview?: { envId: string; sourceId: string };
};

export function GithubServiceLogsDrawer({ open, onOpenChange, projectId, githubServiceId, serviceName, preview }: GithubServiceLogsDrawerProps) {
  const prod = useGithubLogStream(projectId, githubServiceId, open && !preview);
  const prev = usePreviewServiceLogStream(projectId, preview?.envId ?? "", preview?.sourceId ?? "", open && !!preview);
  const { lines, status, clear, firstLineNumber } = preview ? prev : prod;
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
