import { getRuntimeEnv } from "./env.functions";

export interface RuntimeEnv {
  clerkPublishableKey: string;
  apiUrl: string;
}

/** Build-time values baked by Vite (`vite build` + `--build-arg`). Primary source. */
function readBuildEnv(): Partial<RuntimeEnv> {
  return {
    clerkPublishableKey: import.meta.env.VITE_CLERK_PUBLISHABLE_KEY || "",
    apiUrl: (import.meta.env.VITE_API_URL as string | undefined)?.replace(/\/+$/, "") || "",
  };
}

let cache: RuntimeEnv | null = null;

/**
 * Resolve env once at startup: build-time values win, the server function
 * (`GET` → pod env on the Start server, e.g. from k8s Secrets) fills blanks.
 * Throws if anything is still missing — fail fast instead of firing
 * unauthenticated requests at an empty URL.
 */
export async function resolveEnv(): Promise<RuntimeEnv> {
  if (cache) return cache;

  const build = readBuildEnv();
  let runtime: Partial<RuntimeEnv> = {};
  if (!build.clerkPublishableKey || !build.apiUrl) {
    try {
      runtime = await getRuntimeEnv();
    } catch {
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
  cache = merged;
  return merged;
}

/** Synchronous read for API clients / hooks. Only valid after `resolveEnv()`. */
export function getEnv(): RuntimeEnv {
  if (!cache) throw new Error("env not resolved yet — awaiting resolveEnv() first");
  return cache;
}
