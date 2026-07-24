import { createFileRoute } from "@tanstack/react-router";
import { SignUp } from "@clerk/clerk-react";
import { safeRedirectTarget } from "@/lib/safe-redirect";

export const Route = createFileRoute("/sign-up/")({
  component: SignUpPage,
});

function SignUpPage() {
  const { redirect } = Route.useSearch();
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <SignUp
        routing="path"
        path="/sign-up"
        signInUrl="/sign-in"
        forceRedirectUrl={safeRedirectTarget(redirect)}
      />
    </div>
  );
}
