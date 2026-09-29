import type { useAuth } from "@clerk/clerk-react";

export type AuthState = ReturnType<typeof useAuth>;

/**
 * Bridge between Clerk (React) and TanStack Router (beforeLoad).
 *
 * Start owns the <RouterProvider>, so the router can't mount inside
 * <ClerkLoaded> like the old SPA entry did. Instead <AuthSync> (in __root)
 * publishes useAuth() here on every render, root beforeLoad exposes it as
 * router context, and guards read `context.auth` (null = not loaded yet).
 */
export const authRef: { current: AuthState | null } = { current: null };
