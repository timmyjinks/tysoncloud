import { useState } from "react";
import { Copy, Download, Trash2, X } from "lucide-react";
import type { LogStreamStatus } from "@/lib/logs/use-log-stream";
import { LogViewer } from "@/components/log-viewer";
import { StatusDot } from "@/components/status-dot";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type LogsDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  serviceName: string;
  lines: string[];
  status: LogStreamStatus;
  clear: () => void;
  firstLineNumber: number;
};

const STATUS_LABEL: Record<LogStreamStatus, string> = {
  connecting: "connecting",
  open: "live",
  closed: "reconnecting",
  error: "error",
};

export function LogsDrawer({ open, onOpenChange, serviceName, lines, status, clear, firstLineNumber }: LogsDrawerProps) {
  const [autoscroll, setAutoscroll] = useState(true);

  function handleCopy() {
    navigator.clipboard.writeText(lines.join("\n"));
  }

  function handleDownload() {
    const blob = new Blob([lines.join("\n")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${serviceName}-logs.txt`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <>
      <div
        className={cn(
          "fixed inset-0 z-40 bg-black/60 transition-opacity",
          open ? "opacity-100" : "pointer-events-none opacity-0",
        )}
        onClick={() => onOpenChange(false)}
        aria-hidden
      />

      <div
        className={cn(
          "fixed inset-y-0 right-0 z-50 flex w-full max-w-2xl flex-col border-l border-[var(--color-border)] bg-[var(--color-surface)] shadow-2xl transition-transform duration-200",
          open ? "translate-x-0" : "translate-x-full",
        )}
        role="dialog"
        aria-label={`${serviceName} logs`}
      >
        <div className="terminal-strip flex items-center justify-between border-b border-[var(--color-border)] bg-[var(--color-surface-2)] px-5 py-4">
          <div className="flex items-center gap-2 font-mono text-base">
            <span className="text-[var(--color-text)]">{serviceName}</span>
            <span className="text-[var(--color-text-faint)]">· logs</span>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5">
              <StatusDot status={status === "open" ? "running" : "other"} />
              <span className="font-mono text-sm text-[var(--color-text-faint)]">{STATUS_LABEL[status]}</span>
            </div>
            <button
              onClick={() => onOpenChange(false)}
              aria-label="Close logs"
              className="text-[var(--color-text-faint)] hover:text-[var(--color-text)]"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        <div className="flex items-center gap-2 border-b border-[var(--color-border)] px-4 py-2">
          <Button variant="ghost" size="sm" onClick={() => setAutoscroll((v) => !v)}>
            Autoscroll: {autoscroll ? "on" : "off"}
          </Button>
          <Button variant="ghost" size="sm" onClick={handleCopy}>
            <Copy className="h-3.5 w-3.5" />
            Copy
          </Button>
          <Button variant="ghost" size="sm" onClick={handleDownload}>
            <Download className="h-3.5 w-3.5" />
            Download
          </Button>
          <Button variant="ghost" size="sm" onClick={clear}>
            <Trash2 className="h-3.5 w-3.5" />
            Clear
          </Button>
        </div>

        <LogViewer
          lines={lines}
          status={status}
          firstLineNumber={firstLineNumber}
          autoscroll={autoscroll}
          onAutoscrollChange={setAutoscroll}
        />
      </div>
    </>
  );
}
