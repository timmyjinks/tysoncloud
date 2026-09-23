import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import type { useAuth } from "@clerk/clerk-react";

type RouterContext = {
  auth: ReturnType<typeof useAuth>;
};

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
});

function RootLayout() {
  return (
    <div className="min-h-screen bg-[var(--color-bg)] font-sans text-[var(--color-text)]">
      <Outlet />
    </div>
  );
}
