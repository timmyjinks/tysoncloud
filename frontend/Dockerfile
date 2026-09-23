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
RUN pnpm exec vite build

FROM node:22-slim AS runner
WORKDIR /app
RUN npm install -g serve
COPY --from=builder /app/dist ./dist
EXPOSE 3000
CMD ["/bin/sh", "-c", "printf 'window.__ENV__ = {\\n  VITE_CLERK_PUBLISHABLE_KEY: \"%s\",\\n  VITE_API_URL: \"%s\"\\n};\\n' \"$VITE_CLERK_PUBLISHABLE_KEY\" \"$VITE_API_URL\" > /app/dist/env.js && serve -s dist -l 3000"]
