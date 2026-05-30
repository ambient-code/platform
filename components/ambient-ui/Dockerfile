# Use Red Hat UBI Node.js 20 minimal image for dependencies
FROM registry.access.redhat.com/ubi9/nodejs-20-minimal AS deps

WORKDIR /app

USER 0

# Copy and build SDK dependency. package.json references file:../ambient-sdk/ts-sdk
# which resolves to /ambient-sdk/ts-sdk from WORKDIR /app.
COPY ambient-sdk/ts-sdk /ambient-sdk/ts-sdk
RUN cd /ambient-sdk/ts-sdk && npm install --ignore-scripts && npm run build

# Copy ambient-ui package files
COPY ambient-ui/package.json ambient-ui/package-lock.json* ./
RUN npm ci

# Rebuild the source code only when needed
# Use the full nodejs-20 image (not minimal) for the build stage because
# Next.js 16 Turbopack requires native SWC binaries that depend on glibc.
FROM registry.access.redhat.com/ubi9/nodejs-20 AS builder

USER 0

WORKDIR /app

# Copy node_modules and SDK from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /ambient-sdk /ambient-sdk
COPY ambient-ui/ .

# Next.js collects completely anonymous telemetry data about general usage.
# Learn more here: https://nextjs.org/telemetry
ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

# Prepare standalone output with OpenShift-compatible permissions in the builder
# (the hardened runner image is distroless — no shell or chmod available)
# outputFileTracingRoot is set to ../.. (monorepo root), so standalone output
# nests files under components/ambient-ui/ within .next/standalone/
RUN mkdir -p /app-output/public /app-output/.next/static && \
    cp -r .next/standalone/components/ambient-ui/. /app-output/ && \
    cp -r .next/standalone/node_modules /app-output/node_modules 2>/dev/null || true && \
    cp -r .next/static/. /app-output/.next/static/ && \
    cp -r public/. /app-output/public/ && \
    chmod -R g=u /app-output && \
    chgrp -R 0 /app-output

# Production image — Red Hat Hardened Image (distroless, ~48MB, 0 CVEs)
FROM registry.access.redhat.com/hi/nodejs:latest AS runner

ARG GIT_COMMIT=unknown

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

COPY --from=builder /app-output/. ./

USER 1001

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# server.js is created by next build from the standalone output
# https://nextjs.org/docs/pages/api-reference/next-config-js/output
LABEL org.opencontainers.image.revision=$GIT_COMMIT

CMD ["node", "server.js"]
