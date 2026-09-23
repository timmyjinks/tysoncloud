import { getRuntimeEnv } from "./env.functions";

export interface RuntimeEnv {
  clerkPublishableKey: string;
  apiUrl: string;
}

/** Build-time values baked by Vite. Primary source when present. */
function readBuildEnv(): Partial<RuntimeEnv> {
  return {
    clerkPublishableKey: import.meta.env.VITE_CLERK_PUBLISHABLE_KEY || "",
    apiUrl: (import.meta.env.VITE_API_URL as string | undefined)?.replace(/\/+$/, "") || "",
  };
}

let settled: RuntimeEnv | null = null;
let pending: Promise<RuntimeEnv> | null = null;
let failure: unknown = null;

const RUNTIME_FETCH_TIMEOUT_MS = 10_000;

function withTimeout<T>(promise: Promise<T>, ms: number, label: string): Promise<T> {
  let timer: ReturnType<typeof setTimeout>;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => reject(new Error(`${label} timed out after ${ms}ms`)), ms);
  });
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer));
}

/**
 * Resolve env once at startup: build-time values win, the server function
 * (pod env on the Start server, e.g. from k8s Secrets) fills blanks.
 * A hanging server-fn fetch becomes a readable error (never infinite white).
 * Throws if anything is still missing — fail fast instead of firing
 * requests at an empty URL.
 */
export async function resolveEnv(): Promise<RuntimeEnv> {
  if (settled) return settled;

  const build = readBuildEnv();
  const needsRuntime = !build.clerkPublishableKey || !build.apiUrl;
  // Proves the loader ran and whether a runtime fetch was attempted.
  console.info(`[env] resolving (runtime fetch: ${needsRuntime ? "yes" : "not needed"})`);
  let runtime: Partial<RuntimeEnv> = {};
  if (needsRuntime) {
    try {
      runtime = await withTimeout(getRuntimeEnv(), RUNTIME_FETCH_TIMEOUT_MS, "runtime env fetch");
    } catch (err) {
      console.warn("[env] runtime fetch failed, continuing with build-time values:", err);
      runtime = {};
    }
  }

  const merged: RuntimeEnv = {
    clerkPublishableKey: build.clerkPublishableKey || runtime.clerkPublishableKey || "",
    apiUrl: build.apiUrl || (runtime.apiUrl?.replace(/\/+$/, "") ?? "") || "",
  };
  // Logs which layer won per key: build wins, runtime fills blanks.
  console.info(
    `[env] sources: clerk=${build.clerkPublishableKey ? "build" : runtime.clerkPublishableKey ? "runtime" : "MISSING"}, apiUrl=${build.apiUrl ? "build" : runtime.apiUrl ? "runtime" : "MISSING"}`,
  );
  if (!merged.clerkPublishableKey) {
    throw new Error(
      "Missing Clerk publishable key — bake VITE_CLERK_PUBLISHABLE_KEY at build time or set it on the server.",
    );
  }
  if (!merged.apiUrl) {
    throw new Error(
      "Missing API URL — bake VITE_API_URL at build time or set it on the server.",
    );
  }
  settled = merged;
  return merged;
}

/**
 * Render-path read for components. Suspends on first use (safe for the
 * prerendered SPA shell — the Suspense fallback renders instead), throws a
 * readable error on failure, returns the value afterwards.
 */
export function useEnv(): RuntimeEnv {
  if (settled) return settled;
  if (failure) throw failure;
  if (!pending) {
    pending = resolveEnv().then(
      (v) => {
        settled = v;
        return v;
      },
      (e: unknown) => {
        failure = e;
        throw e;
      },
    );
  }
  throw pending;
}

/** Synchronous read for API clients / hooks. Only valid after env resolved. */
export function getEnv(): RuntimeEnv {
  if (!settled) throw new Error("env not resolved yet — awaiting resolveEnv() first");
  return settled;
}
