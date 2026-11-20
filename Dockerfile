# Build: docker build -t penpot-mcp .
# Run (stdio): docker run -i --rm -e PENPOT_ACCESS_TOKEN penpot-mcp
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/penpot-mcp ./cmd/penpot-mcp

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/penpot-mcp /penpot-mcp
ENTRYPOINT ["/penpot-mcp"]
