FROM golang:1.25-alpine AS builder

WORKDIR /src/weather

COPY weather/go.mod weather/go.sum ./
COPY log-lib-go ../log-lib-go
RUN go mod download

COPY weather .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/weather .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/weather /app/weather

EXPOSE 8070

ENTRYPOINT ["/app/weather"]
