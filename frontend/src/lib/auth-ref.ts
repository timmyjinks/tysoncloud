import type { useAuth } from "@clerk/clerk-react";

export type AuthState = ReturnType<typeof useAuth>;

/**
 * Bridge between Clerk (React) and TanStack Router (beforeLoad).
 *
 * Start owns the <RouterProvider>, so unlike the old SPA entry we can't
 * mount the router inside <ClerkLoaded>. Instead <AuthSync> (in __root)
 * publishes useAuth() here on every render, root beforeLoad exposes it as
 * router context, and guards read `context.auth` (null = not loaded yet).
 */
export const authRef: { current: AuthState | null } = { current: null };
