import { createStart, createCsrfMiddleware } from "@tanstack/react-start";

const csrfMiddleware = createCsrfMiddleware({
  filter: (ctx) => ctx.handlerType === "serverFn",
});

export const startInstance = createStart(() => ({
  // SPA mode: no SSR, everything renders on the client like the old Vite
  // SPA. The Node server exists for server functions (runtime envs).
  defaultSsr: false,
  requestMiddleware: [csrfMiddleware],
}));
