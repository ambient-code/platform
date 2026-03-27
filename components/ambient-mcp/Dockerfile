FROM registry.access.redhat.com/ubi9/go-toolset:1.25 AS builder

ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown
ARG GIT_REPO=unknown
ARG GIT_VERSION=unknown
ARG BUILD_DATE=unknown
ARG BUILD_USER=unknown

USER 0
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ambient-mcp .

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

WORKDIR /app

RUN microdnf install -y procps && microdnf clean all

COPY --from=builder /app/ambient-mcp .

RUN chmod +x ./ambient-mcp && chmod 775 /app

USER 1001

ENTRYPOINT ["./ambient-mcp"]
CMD []
