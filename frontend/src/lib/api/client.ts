import type { ApiError } from "./types";

declare global {
  interface Window {
    __ENV__?: Record<string, string>;
  }
}

// window.__ENV__ is written by env.js at container start (see Dockerfile
// CMD) from the pod's actual env — e.g. a k8s secret. import.meta.env is
// baked in at build time, so it's only used as a fallback for local dev
// (`pnpm dev`), where there's no container writing env.js.
const API_URL = window.__ENV__?.VITE_API_URL ?? import.meta.env.VITE_API_URL ?? "";

/**
 * Reads the Clerk session token off the global `window.Clerk` instance.
 * Works both inside React components and inside TanStack Router's
 * beforeLoad/loader, which run outside the React tree and can't call
 * useAuth().
 */
async function getToken(): Promise<string | null> {
  const clerk = (window as any).Clerk;
  if (!clerk?.session) return null;
  return clerk.session.getToken();
}

export class ApiRequestError extends Error {
  status: number;
  body: ApiError | null;

  constructor(status: number, body: ApiError | null, message: string) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

export const NETWORK_ERROR_MESSAGE = "Couldn't reach TYSONCLOUD. Check your connection and try again.";

const STATUS_FALLBACK_MESSAGES: Record<number, string> = {
  401: "Your session has expired. Please sign in again.",
  403: "You don't have permission to do that.",
  404: "We couldn't find what you were looking for.",
  409: "That name is already taken. Please choose a different one.",
  429: "You're moving a little fast — wait a moment and try again.",
};

function statusFallbackMessage(status: number): string {
  if (status >= 500) return "Something went wrong on our end. Please try again.";
  return STATUS_FALLBACK_MESSAGES[status] ?? "That request didn't work as expected.";
}

const NETWORK_ERROR_MESSAGES = new Set([
  "Failed to fetch",
  "NetworkError when attempting to fetch resource.",
  "Load failed",
  "The Internet connection appears to be offline.",
  "Network request failed",
]);

/** Turns any thrown value into a user-facing error message. */
export function getErrorMessage(err: unknown): string {
  if (err instanceof ApiRequestError) return err.message || statusFallbackMessage(err.status);
  if (err instanceof TypeError) {
    return NETWORK_ERROR_MESSAGES.has(err.message) ? NETWORK_ERROR_MESSAGE : err.message;
  }
  if (err instanceof Error) return err.message || "Something unexpected went wrong.";
  if (typeof err === "string") return err;
  return "Something unexpected went wrong.";
}

async function request<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const token = await getToken();

  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}`, {
      ...init,
      headers: {
        ...(init.body ? { "Content-Type": "application/json" } : {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...init.headers,
      },
    });
  } catch {
    throw new ApiRequestError(0, null, NETWORK_ERROR_MESSAGE);
  }

  if (!res.ok) {
    const contentType = res.headers.get("Content-Type") ?? "";
    const text = await res.text().catch(() => "");
    let body: ApiError | null = null;
    let message: string | null = null;

    if (text) {
      if (contentType.includes("application/json")) {
        try {
          body = JSON.parse(text);
          message = body?.error ?? body?.message ?? null;
        } catch {
          body = null;
        }
      } else if (contentType.includes("text/html")) {
        if (text.includes("Service unavailable") || text.includes("isn't reachable")) {
          message = "This service isn't reachable right now. It may still be starting up — wait a moment and try again.";
        } else {
          const title = text.match(/<title>(.*?)<\/title>/i)?.[1]?.trim();
          const h1 = text.match(/<h1[^>]*>(.*?)<\/h1>/i)?.[1]?.trim();
          message = (h1 || title || text).replace(/<[^>]+>/g, " ").replace(/\s+/g, " ").trim().slice(0, 300) || null;
        }
      } else if (contentType.includes("text/plain")) {
        message = text.trim().slice(0, 300) || null;
      }
    }

    throw new ApiRequestError(res.status, body, message ?? statusFallbackMessage(res.status));
  }

  // Several endpoints (CreateService, CreateProject, CreateDatabase, CreateVolume,
  // and the Update* handlers) return a success status with no response body at all —
  // not just 204. Read as text first so an empty body never hits JSON.parse.
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PUT", body: body ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "DELETE", body: body ? JSON.stringify(body) : undefined }),
};
