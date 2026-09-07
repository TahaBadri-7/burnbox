# ---------- build stage ----------
FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /burnbox .

# ---------- runtime stage ----------
FROM alpine:3.22

RUN adduser -D -u 10001 burnbox

COPY --from=build /burnbox /usr/local/bin/burnbox

USER burnbox

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/burnbox"]
