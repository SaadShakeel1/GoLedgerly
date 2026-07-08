# GoLedgerly

GoLedgerly is a staged, hands-on implementation of a distributed key-value store in Go.

## Current milestone

This repository currently implements Stage 0:

- a single-node in-memory store
- an HTTP API for reading, writing, and deleting values
- a mutex-protected store so concurrent requests are safe

## Architecture

The project is organized in the same style as the Let's Go books:

- [cmd/api/main.go](cmd/api/main.go) creates the application, wires dependencies, and starts the server
- [cmd/api/routes.go](cmd/api/routes.go) registers the HTTP routes
- [cmd/api/handlers.go](cmd/api/handlers.go) contains the request handlers
- [internal/store/store.go](internal/store/store.go) contains the core store logic

## Design decisions

- The store is kept separate from the HTTP layer. This keeps the storage logic simple and independent of transport concerns.
- The store uses a mutex because the HTTP server handles requests concurrently in different goroutines.
- The API uses simple HTTP semantics:
  - GET returns the stored value or 404 if it does not exist
  - PUT stores a value and returns 201
  - DELETE removes a value and returns 204

## Running the app

Start the server:

```bash
go run ./cmd/api
```

Example requests:

```bash
curl http://localhost:4000/health
curl -X PUT http://localhost:4000/store/hello -d "world"
curl http://localhost:4000/store/hello
curl -X DELETE http://localhost:4000/store/hello
```

## Testing

Run the test suite:

```bash
go test ./...
```
