FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod .
COPY internal ./internal
COPY stacks/slopper/gateway ./stacks/slopper/gateway
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /gateway ./stacks/slopper/gateway
FROM scratch
COPY --from=build /gateway /gateway
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/gateway"]
