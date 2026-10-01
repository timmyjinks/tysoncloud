import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { safeRedirectTarget } from "@/lib/safe-redirect";

type SignInSearch = { redirect?: string };

export const Route = createFileRoute("/sign-in")({
  validateSearch: (search: Record<string, unknown>): SignInSearch => ({
    redirect: typeof search.redirect === "string" ? search.redirect : undefined,
  }),
  beforeLoad: ({ context, search }) => {
    if (context.auth.isSignedIn) {
      throw redirect({ to: safeRedirectTarget(search.redirect) });
    }
  },
  component: Outlet,
});
