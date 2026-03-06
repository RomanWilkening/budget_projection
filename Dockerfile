# ---- Stage 1: Build Frontend ----
FROM node:22-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: Build Backend ----
FROM golang:1.25-alpine AS backend-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./

# Copy built frontend into the backend build context
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

RUN CGO_ENABLED=1 GOOS=linux go build -o /app/server .

# ---- Stage 3: Runtime ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /app/server .
COPY --from=backend-builder /app/backend/frontend/dist ./frontend/dist

# Create data directory for SQLite
RUN mkdir -p /app/data

ENV DB_PATH=/app/data/budget.db
ENV PORT=8080
ENV GIN_MODE=release

EXPOSE 8080

VOLUME ["/app/data"]

CMD ["./server"]
