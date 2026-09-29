# Amvera production build (build context = repo root).
# Canonical service sources live in kanban/; keep in sync with kanban/Dockerfile.
FROM node:20-alpine AS frontend-build
WORKDIR /app
COPY kanban/frontend/package*.json ./frontend/
RUN cd frontend && npm ci
COPY kanban/frontend/ ./frontend/
RUN cd frontend && npm run build

FROM golang:1.26-alpine AS go-build
WORKDIR /app
COPY kanban/go.mod kanban/go.sum ./
RUN go mod download
COPY kanban/ .
COPY --from=frontend-build /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /taskflow .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=go-build /taskflow .
COPY kanban/db/migrations ./db/migrations

EXPOSE 9300
CMD ["./taskflow"]
