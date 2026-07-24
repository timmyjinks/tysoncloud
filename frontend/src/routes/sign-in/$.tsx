import { createFileRoute } from "@tanstack/react-router";
import { SignIn } from "@clerk/clerk-react";
import { safeRedirectTarget } from "@/lib/safe-redirect";

// Clerk's <SignIn routing="path"> pushes sub-steps like /sign-in/factor-one,
// /sign-in/sso-callback, /sign-in/reset-password, etc. It reads the URL
// itself to know which step to render, so this route just needs to exist
// and mount the same component — the index route (./index.tsx) handles the
// bare /sign-in path, this splat route catches everything under it.
export const Route = createFileRoute("/sign-in/$")({
  component: SignInStepPage,
});

function SignInStepPage() {
  const { redirect } = Route.useSearch();
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <SignIn
        routing="path"
        path="/sign-in"
        signUpUrl="/sign-up"
        forceRedirectUrl={safeRedirectTarget(redirect)}
      />
    </div>
  );
}
