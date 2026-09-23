import { useEffect, useState } from "react";
import { getRuntimeEnv, type PublicRuntimeEnv } from "./env.functions";

export interface RuntimeEnv {
  clerkPublishableKey: string;
  apiUrl: string;
}

let current: RuntimeEnv | null = null;

/** Validates + stores server data. Throws a readable error if incomplete. */
export function setEnv(env: PublicRuntimeEnv): RuntimeEnv {
  const normalized: RuntimeEnv = {
    clerkPublishableKey: env.clerkPublishableKey.trim(),
    apiUrl: env.apiUrl.trim().replace(/\/+$/, ""),
  };
  if (!normalized.clerkPublishableKey) {
    throw new Error(
      "Missing Clerk publishable key — set VITE_CLERK_PUBLISHABLE_KEY on the server.",
    );
  }
  if (!normalized.apiUrl) {
    throw new Error("Missing API URL — set VITE_API_URL on the server.");
  }
  current = normalized;
  return normalized;
}

/** Synchronous read for API clients / hooks. Only valid after env has loaded. */
export function getEnv(): RuntimeEnv {
  if (!current) throw new Error("env not loaded yet — root loader must run first");
  return current;
}

/**
 * Documented pattern: fetch runtime env from the server after mount.
 * Never runs during SSR/prerender (effects don't run on the server), so the
 * static shell builds without secrets and the browser always gets live values.
 */
export function useRuntimeEnv(): { env: RuntimeEnv | null; error: unknown } {
  const [state, setState] = useState<{ env: RuntimeEnv | null; error: unknown }>({
    env: current,
    error: null,
  });

  useEffect(() => {
    if (current) {
      setState({ env: current, error: null });
      return;
    }
    let cancelled = false;
    getRuntimeEnv().then(
      (data) => {
        if (cancelled) return;
        try {
          setState({ env: setEnv(data), error: null });
        } catch (e) {
          setState({ env: null, error: e });
        }
      },
      (e: unknown) => {
        if (!cancelled) setState({ env: null, error: e });
      },
    );
    return () => {
      cancelled = true;
    };
  }, []);

  return state;
}
