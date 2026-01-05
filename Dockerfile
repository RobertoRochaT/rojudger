# Build stage
FROM golang:1.21-alpine AS builder

# Instalar dependencias necesarias para compilar
RUN apk add --no-cache git gcc musl-dev

# Establecer directorio de trabajo
WORKDIR /build

# Copiar go.mod y go.sum primero (mejor cache de Docker)
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar todo el código fuente
COPY . .

# Compilar aplicación (con CGO para PostgreSQL)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o rojudger-api ./cmd/api

# Runtime stage
FROM alpine:3.18

# Instalar dependencias de runtime
RUN apk --no-cache add ca-certificates docker-cli wget

# Crear directorio de trabajo
WORKDIR /app

# Copiar binario compilado desde build stage
COPY --from=builder /build/rojudger-api /app/rojudger-api

# Hacer el binario ejecutable
RUN chmod +x /app/rojudger-api

# Exponer puerto
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Comando de inicio
CMD ["/app/rojudger-api"]
