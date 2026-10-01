import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { safeRedirectTarget } from "@/lib/safe-redirect";

type SignUpSearch = { redirect?: string };

export const Route = createFileRoute("/sign-up")({
  validateSearch: (search: Record<string, unknown>): SignUpSearch => ({
    redirect: typeof search.redirect === "string" ? search.redirect : undefined,
  }),
  beforeLoad: ({ context, search }) => {
    if (context.auth.isSignedIn) {
      throw redirect({ to: safeRedirectTarget(search.redirect) });
    }
  },
  component: Outlet,
});
