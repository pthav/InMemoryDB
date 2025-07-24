# Base image
FROM golang:1.24.5-bookworm AS base
LABEL authors="pthav"

# Development Stage
# =============================================================================
# Create development stage from the base image
FROM base AS development

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

# Build Stage
# =============================================================================
# Create a build stage from the base image
FROM base AS build

WORKDIR /build

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o db

# Production Stage
# =============================================================================
# Create a production stage to run the binary
FROM scratch AS prod

WORKDIR /prod

COPY --from=build /build/db ./

EXPOSE 8080

CMD ["./db","server", "serve", "--host", "0.0.0.0:8080"]