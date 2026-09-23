import { createServerFn } from "@tanstack/react-start";

export interface PublicRuntimeEnv {
  clerkPublishableKey: string;
  apiUrl: string;
}

/**
 * Runtime env, served by the Start Node server.
 *
 * `process.env` is read per-request inside `.handler()` — never at module
 * scope (module-scope reads leak into the client bundle and are `undefined`
 * on edge runtimes). Only the two public values cross to the client; real
 * secrets must never be added to this return.
 *
 * Empty string means "not configured here" — the client merges this with
 * the build-time `import.meta.env` values and validates the result.
 * VITE_* names are read first so existing k8s Secrets keep working as-is.
 */
export const getRuntimeEnv = createServerFn({ method: "GET" }).handler(
  async (): Promise<PublicRuntimeEnv> => {
    const clerkPublishableKey = (
      process.env.VITE_CLERK_PUBLISHABLE_KEY ??
      process.env.CLERK_PUBLISHABLE_KEY ??
      ""
    ).trim();
    const apiUrl = (
      process.env.VITE_API_URL ??
      process.env.API_URL ??
      ""
    ).trim();
    // Logs key presence (never values): proves pod env reaches the server.
    console.log(
      `[env] runtime keys present: clerk=${clerkPublishableKey ? `yes(len=${clerkPublishableKey.length})` : "NO"}, apiUrl=${apiUrl ? `yes(${apiUrl})` : "NO"}`,
    );
    return { clerkPublishableKey, apiUrl };
  },
);
