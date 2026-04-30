FROM golang:1.26-bookworm AS build

WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/rayip-api ./services/api/cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/rayip-api /rayip-api
EXPOSE 8080 9090
ENTRYPOINT ["/rayip-api"]
