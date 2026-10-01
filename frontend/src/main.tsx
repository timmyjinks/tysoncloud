import React from "react";
import ReactDOM from "react-dom/client";
import { ClerkProvider, ClerkLoaded, ClerkLoading, useAuth } from "@clerk/clerk-react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";
import "./app.css";

declare global {
  interface Window {
    __ENV__?: Record<string, string>;
  }
}

const PUBLISHABLE_KEY =
  window.__ENV__?.VITE_CLERK_PUBLISHABLE_KEY || import.meta.env.VITE_CLERK_PUBLISHABLE_KEY;
if (!PUBLISHABLE_KEY) {
  throw new Error("Missing VITE_CLERK_PUBLISHABLE_KEY — copy .env.example to .env.local");
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
});

const router = createRouter({
  routeTree,
  context: { auth: undefined! }, // populated by InnerApp below, once Clerk has loaded
  defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

function InnerApp() {
  const auth = useAuth();
  return <RouterProvider router={router} context={{ auth }} />;
}

function FullPageLoading() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-[var(--color-bg)]">
      <p className="font-mono text-base text-[var(--color-text-faint)]">loading…</p>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ClerkProvider
      publishableKey={PUBLISHABLE_KEY}
      signInFallbackRedirectUrl="/dashboard"
      signUpFallbackRedirectUrl="/dashboard"
      afterSignOutUrl="/"
      routerPush={(to) => router.navigate({ to: to as any, replace: false })}
      routerReplace={(to) => router.navigate({ to: to as any, replace: true })}
      appearance={{
        variables: {
          colorPrimary: "#ff4433",
          colorBackground: "#17140f",
          colorInputBackground: "#1d1913",
          colorInputText: "#f4ede1",
          colorText: "#f4ede1",
          colorTextSecondary: "#b5ac9d",
          colorDanger: "#ff4433",
          colorSuccess: "#a3b98a",
          colorWarning: "#e8b356",
          borderRadius: "0.375rem",
          fontFamily: "'IBM Plex Sans', ui-sans-serif, system-ui, sans-serif",
        },
        elements: {
          badge: {
            backgroundColor: "#1d1913",
            color: "#b5ac9d",
            border: "1px solid #2a261f",
          },
          formFieldInput: {
            backgroundColor: "#1d1913",
            border: "1px solid #3a352b",
          },
          formButtonPrimary: {
            backgroundColor: "#ff4433",
            ":hover": {
              backgroundColor: "#e23a2c",
            },
          },
          userButtonPopoverCard: {
            backgroundColor: "#17140f",
            border: "1px solid #2a261f",
          },
          userButtonPopoverActionButton: {
            color: "#f4ede1",
            "&:hover": {
              backgroundColor: "#242019",
              color: "#f4ede1",
            },
            "&:focus": {
              color: "#f4ede1",
            },
          },
          userButtonPopoverActionButtonText: {
            color: "#f4ede1",
            "&:hover": {
              color: "#f4ede1",
            },
          },
          userButtonPopoverActionButtonIcon: {
            color: "#b5ac9d",
            "&:hover": {
              color: "#b5ac9d",
            },
          },
          userButtonPopoverFooter: {
            backgroundColor: "#1d1913",
          },
          userButtonTrigger: {
            "&:hover": {
              backgroundColor: "#242019",
            },
            "&:focus": {
              boxShadow: "none",
            },
          },
          formFieldInputShowPasswordButton: {
            color: "#b5ac9d",
            "&:hover": {
              backgroundColor: "#242019",
              color: "#f4ede1",
            },
          },
          avatarImageActionsUpload: {
            backgroundColor: "#ff4433",
            color: "#17140f",
            "&:hover": {
              backgroundColor: "#e23a2c",
            },
            "&:focus": {
              backgroundColor: "#e23a2c",
            },
          },
          avatarImageActionsUploadInDropArea: {
            backgroundColor: "#ff4433",
            color: "#17140f",
            "&:hover": {
              backgroundColor: "#e23a2c",
            },
          },
          avatarImageActionsDownload: {
            backgroundColor: "#1d1913",
            color: "#f4ede1",
            "&:hover": {
              backgroundColor: "#242019",
            },
          },
          avatarImageActionsRemove: {
            backgroundColor: "#1d1913",
            color: "#ff4433",
            "&:hover": {
              backgroundColor: "#242019",
            },
          },
        },
      }}
    >
      <QueryClientProvider client={queryClient}>
        <ClerkLoading>
          <FullPageLoading />
        </ClerkLoading>
        <ClerkLoaded>
          <InnerApp />
        </ClerkLoaded>
      </QueryClientProvider>
    </ClerkProvider>
  </React.StrictMode>,
);
