import { useLogStream, usePreviewServiceLogStream } from "@/lib/logs/use-log-stream";
import { LogsDrawer } from "@/components/logs-drawer";

type ServiceLogsDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
  serviceId: string;
  serviceName: string;
  /** When present, streams the preview copy's logs instead of prod logs. */
  preview?: { envId: string; sourceId: string };
};

export function ServiceLogsDrawer({ open, onOpenChange, projectId, serviceId, serviceName, preview }: ServiceLogsDrawerProps) {
  const prod = useLogStream(projectId, serviceId, open && !preview);
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
