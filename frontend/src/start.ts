import { createStart } from "@tanstack/react-start";

export const startInstance = createStart(() => ({
  // SPA mode: no SSR, everything renders on the client exactly like the old
  // Vite SPA. The Node server exists for server functions (runtime envs, …).
  defaultSsr: false,
}));
