import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { Navbar } from "@/components/navbar";

export const Route = createFileRoute("/projects/$projectId")({
  beforeLoad: ({ context, location }) => {
    if (!context.auth.isSignedIn) {
      throw redirect({ to: "/sign-in", search: { redirect: location.href } });
    }
  },
  component: ProjectLayout,
});

function ProjectLayout() {
  return (
    <div>
      <Navbar />
      <Outlet />
    </div>
  );
}
