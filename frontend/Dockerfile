FROM node:22-slim AS deps
WORKDIR /app
RUN corepack enable
COPY pnpm-lock.yaml package.json pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

FROM node:22-slim AS builder
WORKDIR /app
RUN corepack enable
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# Build-time public env, baked into the client bundle via import.meta.env.
# Anything left blank here is resolved at runtime by the Start server
# (src/lib/env.functions.ts) from the container's env — e.g. k8s secrets.
ARG VITE_CLERK_PUBLISHABLE_KEY=""
ARG VITE_API_URL=""
ENV VITE_CLERK_PUBLISHABLE_KEY=$VITE_CLERK_PUBLISHABLE_KEY \
    VITE_API_URL=$VITE_API_URL
RUN pnpm exec vite build

FROM node:22-slim AS runner
WORKDIR /app
RUN corepack enable
ENV NODE_ENV=production \
    PORT=3000
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/package.json /app/pnpm-lock.yaml /app/pnpm-workspace.yaml ./
RUN pnpm prune --prod
COPY --from=builder /app/.output ./output
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
