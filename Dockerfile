# ---- Builder stage ----
FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/bot .

# ---- Runtime stage ----
FROM linuxserver/calibre:latest

# Non-root user provided by the base image (abc).
# Create a writable directory for downloaded/uploaded files.
RUN mkdir -p /files && chown -R abc:abc /files

COPY --from=builder /out/bot /usr/local/bin/bot

WORKDIR /files

USER abc

CMD ["bot"]
