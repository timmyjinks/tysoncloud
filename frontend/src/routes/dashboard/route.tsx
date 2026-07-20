import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { Navbar } from "@/components/navbar";

export const Route = createFileRoute("/dashboard")({
  beforeLoad: ({ context, location }) => {
    if (!context.auth.isSignedIn) {
      throw redirect({ to: "/sign-in", search: { redirect: location.href } });
    }
  },
  component: DashboardLayout,
});

function DashboardLayout() {
  return (
    <div>
      <Navbar />
      <Outlet />
    </div>
  );
}
