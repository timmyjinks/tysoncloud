import {
  HeadContent,
  Outlet,
  Scripts,
  createRootRouteWithContext,
  useRouter,
  type ErrorComponentProps,
} from "@tanstack/react-router";
import {
  ClerkLoaded,
  ClerkLoading,
  ClerkProvider,
  useAuth,
} from "@clerk/clerk-react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React, { useEffect } from "react";
import { authRef, type AuthState } from "@/lib/auth-ref";
import { useRuntimeEnv } from "@/lib/env";

type RouterContext = {
  auth: AuthState | null;
};

export const Route = createRootRouteWithContext<RouterContext>()({
  // Document shell (always rendered, incl. SPA shell prerender).
  shellComponent: RootDocument,
  component: RootComponent,
  beforeLoad: () => ({ auth: authRef.current }),
  errorComponent: EnvError,
});

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
});

/** Publishes Clerk state for beforeLoad guards, then re-runs them once loaded. */
function AuthSync() {
  const auth = useAuth();
  const router = useRouter();
  authRef.current = auth;
  useEffect(() => {
    if (auth.isLoaded) void router.invalidate();
  }, [auth.isLoaded, router]);
  return null;
}

function RootComponent() {
  // Fetched client-side after mount: never runs during SSR/prerender, so the
  // shell builds without secrets and the browser always gets live server env.
  const { env, error } = useRuntimeEnv();
  const router = useRouter();
  if (error) return <EnvError error={error} reset={() => window.location.reload()} />;
  if (!env) return <FullPageLoading />;
  return (
    <ClerkErrorBoundary>
      <ClerkProvider
        publishableKey={env.clerkPublishableKey}
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
            <AuthSync />
            <div className="min-h-screen bg-[var(--color-bg)] font-sans text-[var(--color-text)]">
              <Outlet />
            </div>
          </ClerkLoaded>
        </QueryClientProvider>
      </ClerkProvider>
    </ClerkErrorBoundary>
  );
}

function FullPageLoading() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-[var(--color-bg)]">
      <p className="font-mono text-base text-[var(--color-text-faint)]">loading…</p>
    </div>
  );
}

/** A bad env or Clerk key shows an error, never a white screen. */
class ClerkErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { error: unknown }
> {
  state = { error: null as unknown };
  static getDerivedStateFromError(error: unknown) {
    return { error };
  }
  render() {
    if (this.state.error) return <EnvError error={this.state.error} reset={() => {}} />;
    return this.props.children;
  }
}

function EnvError({ error }: ErrorComponentProps) {
  const message =
    error instanceof Error ? error.message : "Failed to load configuration.";
  return (
    <div className="flex min-h-screen items-center justify-center bg-[var(--color-bg)]">
      <p className="max-w-md px-4 text-center font-mono text-sm text-[var(--color-text-muted)]">
        {message}
      </p>
    </div>
  );
}

const ORG_JSON_LD = `{
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "Organization",
      "name": "TYSONCLOUD",
      "url": "https://tysoncloud.dev/"
    },
    {
      "@type": "WebSite",
      "name": "TYSONCLOUD",
      "url": "https://tysoncloud.dev/"
    },
    {
      "@type": "SoftwareApplication",
      "name": "TYSONCLOUD",
      "applicationCategory": "DeveloperApplication",
      "operatingSystem": "Web",
      "url": "https://tysoncloud.dev/",
      "offers": {
        "@type": "Offer",
        "price": "0",
        "priceCurrency": "USD"
      }
    }
  ]
}`;

const FAQ_JSON_LD = `{
  "@context": "https://schema.org",
  "@type": "FAQPage",
  "mainEntity": [
    {
      "@type": "Question",
      "name": "What is TYSONCLOUD?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "TYSONCLOUD is a deploy platform. Push a Docker image and get a running service, a public domain with automatic TLS, and a managed Postgres database if you need one — no servers to babysit."
      }
    },
    {
      "@type": "Question",
      "name": "How do I deploy a service?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Create a project, add a service with a Docker image and a port, optionally set environment variables, and TYSONCLOUD builds, runs, and exposes it on a tysoncloud.dev domain."
      }
    },
    {
      "@type": "Question",
      "name": "What databases are supported?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Managed Postgres today. Each database gets an internal hostname and storage you can scale from the dashboard."
      }
    },
    {
      "@type": "Question",
      "name": "Can I attach persistent storage?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Yes. Attach a volume to any service with a mount path and storage size, and it survives redeploys."
      }
    },
    {
      "@type": "Question",
      "name": "How do I define infrastructure without clicking forms?",
      "acceptedAnswer": {
        "@type": "Answer",
        "text": "Use config as code: declare services, databases, and volumes in TOML and apply them to a project in one shot."
      }
    }
  ]
}`;

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>TYSONCLOUD</title>
        <meta
          name="description"
          content="TYSONCLOUD is a deploy platform. Push a Docker image, get a running service, a public domain with TLS, and managed Postgres — without managing servers."
        />
        <link rel="canonical" href="https://tysoncloud.dev/" />
        <meta name="robots" content="index, follow" />
        <meta name="theme-color" content="#100e0b" />
        <link rel="icon" type="image/svg+xml" href="/favicon.svg" />

        <meta property="og:type" content="website" />
        <meta property="og:site_name" content="TYSONCLOUD" />
        <meta property="og:title" content="TYSONCLOUD — Deploy services, databases, and volumes in one shot" />
        <meta
          property="og:description"
          content="TYSONCLOUD is a deploy platform. Push a Docker image, get a running service, a public domain with TLS, and managed Postgres — without managing servers."
        />
        <meta property="og:url" content="https://tysoncloud.dev/" />
        <meta property="og:locale" content="en_US" />

        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:title" content="TYSONCLOUD — Deploy services, databases, and volumes in one shot" />
        <meta
          name="twitter:description"
          content="TYSONCLOUD is a deploy platform. Push a Docker image, get a running service, a public domain with TLS, and managed Postgres — without managing servers."
        />

        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />

        <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: ORG_JSON_LD }} />
        <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: FAQ_JSON_LD }} />
        <HeadContent />
      </head>
      <body>
        {children}
        <noscript>
          <p
            style={{
              margin: 0,
              padding: "2rem",
              background: "#100e0b",
              color: "#b5ac9d",
              fontFamily: "ui-monospace, monospace",
              fontSize: "14px",
              textAlign: "center",
            }}
          >
            TYSONCLOUD requires JavaScript to deploy services and databases.
          </p>
        </noscript>
        <Scripts />
      </body>
    </html>
  );
}
