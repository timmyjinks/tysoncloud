import { useRef, useState } from "react";
import { useUpdatePreviewService } from "@/lib/api/previews";
import { getErrorMessage } from "@/lib/api/client";
import { formatEnvLines } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ErrorBanner } from "@/components/error-banner";

export type PreviewEditTarget = {
  sourceId: string;
  serviceName: string;
  env: Record<string, string>;
  port: number;
} | null;

type PreviewServiceEditDialogProps = {
  projectId: string;
  envId: string;
  service: PreviewEditTarget;
  onClose: () => void;
  onSaved?: () => void;
};

/**
 * Port + env only editor for preview copies. Previews are ephemeral and
 * managed by the pull request — name/image/domain/branch are intentionally
 * not editable here. Saving goes through the preview endpoint which applies
 * to this preview only and resets on the next sync.
 */
export function PreviewServiceEditDialog({
  projectId,
  envId,
  service,
  onClose,
  onSaved,
}: PreviewServiceEditDialogProps) {
  const update = useUpdatePreviewService(projectId, envId);
  const [envText, setEnvText] = useState("");
  const [portText, setPortText] = useState("");
  const serviceKey = service?.sourceId ?? "";
  const lastKey = useRef("");
  if (service && lastKey.current !== serviceKey) {
    lastKey.current = serviceKey;
    setEnvText(formatEnvLines(service.env ?? {}));
    setPortText(String(service.port ?? ""));
  }

  const initialEnv = service ? formatEnvLines(service.env ?? {}) : "";
  const initialPort = service ? String(service.port ?? "") : "";
  const portNum = Number(portText);
  const validPort =
    portText.trim() !== "" && Number.isInteger(portNum) && portNum >= 1 && portNum <= 65535;
  const changed = service != null && (envText !== initialEnv || portText !== initialPort);

  return (
    <Dialog
      open={service != null}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
          update.reset();
        }
      }}
    >
      <DialogContent>
        <h3 className="text-lg font-medium text-[var(--color-text)]">
          Update {service?.serviceName ?? "preview service"}
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-muted)]">
          Applies to this preview only — the domain can&apos;t be changed here, and edits
          reset on the next PR sync.
        </p>
        <form
          className="mt-5 space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            if (!service || !changed || !validPort || update.isPending) return;
            const body: { env?: string; port?: number } = {};
            if (envText !== initialEnv) body.env = envText;
            if (portText !== initialPort) body.port = portNum;
            update.mutate(
              { sourceId: service.sourceId, body },
              {
                onSuccess: () => {
                  onClose();
                  onSaved?.();
                },
              },
            );
          }}
        >
          <div>
            <Label htmlFor="preview-port">Port</Label>
            <Input
              id="preview-port"
              type="number"
              required
              value={portText}
              onChange={(e) => setPortText(e.target.value)}
              className="mt-2 font-mono"
            />
          </div>
          <div>
            <Label htmlFor="preview-env">Environment variables</Label>
            <Textarea
              id="preview-env"
              value={envText}
              onChange={(e) => setEnvText(e.target.value)}
              placeholder={"KEY=value\nANOTHER_KEY=value"}
              rows={6}
              className="mt-2 font-mono"
            />
            <p className="mt-1 text-xs text-[var(--color-text-muted)]">
              One <code>KEY=value</code> pair per line. Saving replaces the full set of
              environment variables with what&apos;s shown here.
            </p>
          </div>
          {update.error && (
            <ErrorBanner message={getErrorMessage(update.error)} />
          )}
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button size="sm" type="submit" disabled={!changed || !validPort || update.isPending}>
              {update.isPending ? "Saving…" : "Save changes"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
