FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/weather .

FROM gcr.io/distroless/static-debian12:nonroot

ENV CI=true
WORKDIR /app

COPY --from=builder /out/weather /app/weather

EXPOSE 8070

ENTRYPOINT ["/app/weather"]
