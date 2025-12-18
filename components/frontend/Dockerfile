# Use Red Hat UBI Node.js 20 minimal image for dependencies
FROM registry.access.redhat.com/ubi9/nodejs-20-minimal AS deps

# Build arguments for metadata
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

WORKDIR /app

USER 0

# Install dependencies based on the preferred package manager
COPY package.json package-lock.json* ./
RUN npm ci

# Rebuild the source code only when needed
FROM registry.access.redhat.com/ubi9/nodejs-20-minimal AS builder

# Build arguments (need to redeclare for each stage)
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

USER 0

WORKDIR /app

# Copy node_modules from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Next.js collects completely anonymous telemetry data about general usage.
# Learn more here: https://nextjs.org/telemetry
# Uncomment the following line in case you want to disable telemetry during the build.
ENV NEXT_TELEMETRY_DISABLED=1

# Make build metadata available to Next.js at build time
ENV NEXT_PUBLIC_GIT_COMMIT=${GIT_COMMIT}
ENV NEXT_PUBLIC_GIT_BRANCH=${GIT_BRANCH}
ENV NEXT_PUBLIC_GIT_REPO=${GIT_REPO}
ENV NEXT_PUBLIC_GIT_VERSION=${GIT_VERSION}
ENV NEXT_PUBLIC_BUILD_DATE=${BUILD_DATE}
ENV NEXT_PUBLIC_BUILD_USER=${BUILD_USER}

RUN npm run build

# Production image, copy all the files and run next
FROM registry.access.redhat.com/ubi9/nodejs-20-minimal AS runner

# Build arguments (need to redeclare for final stage)
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

# Add labels to force cache invalidation and provide metadata
LABEL git.commit="${GIT_COMMIT}"
LABEL git.branch="${GIT_BRANCH}"
LABEL git.version="${GIT_VERSION}"
LABEL build.date="${BUILD_DATE}"
LABEL build.user="${BUILD_USER}"

WORKDIR /app

ENV NODE_ENV=production
# Uncomment the following line in case you want to disable telemetry during runtime.
ENV NEXT_TELEMETRY_DISABLED=1

# Build metadata as environment variables (NEXT_PUBLIC_ prefix makes them available to client)
ENV NEXT_PUBLIC_GIT_COMMIT=${GIT_COMMIT}
ENV NEXT_PUBLIC_GIT_BRANCH=${GIT_BRANCH}
ENV NEXT_PUBLIC_GIT_REPO=${GIT_REPO}
ENV NEXT_PUBLIC_GIT_VERSION=${GIT_VERSION}
ENV NEXT_PUBLIC_BUILD_DATE=${BUILD_DATE}
ENV NEXT_PUBLIC_BUILD_USER=${BUILD_USER}

# Copy public assets
COPY --from=builder /app/public ./public

USER 0

# Automatically leverage output traces to reduce image size
# https://nextjs.org/docs/advanced-features/output-file-tracing
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

# Create directories and set permissions for OpenShift arbitrary UIDs
# OpenShift runs containers with random UIDs in the root group (GID 0)
# chmod g=u gives the root group the same permissions as the owner
RUN mkdir -p .next && \
    chmod -R g=u /app && \
    chgrp -R 0 /app

USER 1001

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# server.js is created by next build from the standalone output
# https://nextjs.org/docs/pages/api-reference/next-config-js/output
CMD ["node", "server.js"]
