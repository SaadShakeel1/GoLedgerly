# GoLedgerly — Handoff Context

## What this project is
A staged, from-scratch build of a distributed key-value store in Go, done as a learning/interview-prep project. Built stage by stage, in parallel with reading **Let's Go** and **Let's Go Further** (Alex Edwards). The full staged plan (Stage 0 through Stage 5 + optional extensions) lives in the original build guide the user is following — ask the user for it if you need the long-term roadmap; the summary below covers what's relevant now.

**Stages, in order:**
0. Single-node in-memory cache with HTTP API (mutex-protected)
1. TTL expiration + O(1) LRU eviction
2. Persistence via write-ahead log (WAL), replay on startup, graceful shutdown
3. Production concerns: token-bucket rate limiting, metrics endpoint, health check, full test suite, benchmarks
4. Leader-follower replication across nodes
5. Raft consensus (optional, ambitious)

## How the user wants to work — READ THIS FIRST
This is the most important section. The user is treating this as a learning exercise, not a delivery task.

- **Mentor, don't deliver.** Guide with concepts, "why", and gotchas. Do not silently write large chunks of finished code and hand it over as if the task is done.
- **The user is a first-time Go developer / first real project.** They don't yet know Go syntax. Because of this, it's been agreed that showing **small code snippets in chat** (not written directly to files by the assistant) is fine — the user types them in personally, runs them, and asks questions if stuck. This was a deliberate adjustment from a stricter "concepts only, no code at all" mode, because pure concept-guidance was too abstract for someone who doesn't know syntax yet.
- **Hard boundary: never create/edit/delete files in the user's project directly.** The user runs every command and writes/saves every file themselves. Assistant-side tools should only be used to *read* files for review, not to write them — even for "harmless" scaffolding.
- **Explain every command given** — flags, side effects, why it's needed — not just paste a command line.
- **Follow Let's Go / Let's Go Further conventions from the start, not as an afterthought.** Concretely: an `application` struct in `main.go` holding shared dependencies (store, logger, config, etc.), with handlers implemented as **methods** on `*application` living in a separate `handlers.go` file in the same package. `main.go` should only build dependencies, wire routes, and start the server — it should not contain inline handler logic. (Earlier in this project the assistant initially gave inline anonymous-closure handlers in `main.go` and had to be corrected into this pattern after the fact — don't repeat that; apply it proactively from the first snippet.)
- **Reinforce the guide's engineering habits:** small frequent commits with meaningful messages, a README that explains *why* decisions were made (not just how to run it), tests written alongside each behavior (not batched for later), and running `go run -race` after any change touching shared/concurrent state.
- **When reviewing code the user wrote:** point out issues and explain them, don't rewrite wholesale.
- **Pacing:** user has ~15 hrs/week available. Realistic target discussed: finish Stage 0 this week, Stage 1 (TTL+LRU) week 2, Stage 2 (WAL) week 3, solid start on Stage 3 by end of week 4 — full Stage 3 polish (tests+benchmarks) may spill past the month, which is fine. LRU and WAL are flagged as the two spots most likely to need extra time since they're genuinely new concepts, not just Go syntax — that's expected, not a sign of falling behind.

## Current project structure
```
GoLedgerly/
├── go.mod                  # module github.com/SaadShakeel1/goledgerly
├── cmd/
│   └── api/
│       ├── main.go         # application struct, dependency wiring, route registration, server start
│       └── handlers.go     # handler methods on *application
└── internal/
    └── store/
        └── store.go        # Store type wrapping map[string]string — NO mutex yet
```

Module path: `github.com/SaadShakeel1/goledgerly`. GitHub repo exists and is connected (`origin` remote set, pushed to `main`).

## Current file contents (as of handoff)

**go.mod**
```
module github.com/SaadShakeel1/goledgerly

go 1.26.4
```

**internal/store/store.go**
```go
package store

type Store struct {
	data map[string]string
}

func New() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Get(key string) (string, bool) {
	value, found := s.data[key]
	return value, found
}

func (s *Store) Set(key, value string) {
	s.data[key] = value
}

func (s *Store) Delete(key string) {
	delete(s.data, key)
}
```

**cmd/api/main.go**
```go
package main

import (
	"log"
	"net/http"

	"github.com/SaadShakeel1/goledgerly/internal/store"
)

type application struct {
	store *store.Store
}

func main() {
	app := &application{
		store: store.New(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", app.healthHandler)
	mux.HandleFunc("GET /store/{key}", app.getHandler)
	mux.HandleFunc("PUT /store/{key}", app.putHandler)
	mux.HandleFunc("DELETE /store/{key}", app.deleteHandler)

	log.Println("starting server on :4000")
	err := http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}
```

**cmd/api/handlers.go**
```go
package main

import (
	"io"
	"net/http"
)

func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (app *application) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value, found := app.store.Get(key)
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))
}

func (app *application) putHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	app.store.Set(key, string(body))
	w.WriteHeader(http.StatusCreated)
}

func (app *application) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	app.store.Delete(key)
	w.WriteHeader(http.StatusNoContent)
}
```

## Git state at handoff
- Commits so far:
  - `c3c95f4` — init: go module, project scaffolding
  - `a652e83` — feat: wire in-memory store to HTTP GET/PUT/DELETE routes
- **Not yet committed:** the `application` struct / `handlers.go` refactor. `git status` shows `cmd/api/main.go` modified and `cmd/api/handlers.go` untracked. This refactor still needs a commit, e.g. `refactor: extract handlers into application struct pattern`.
- Remote `origin` is set to the user's GitHub repo (`SaadShakeel1/goledgerly`) and `main` is pushed up through the second commit.

## Stage 0 progress
Done:
1. Bare HTTP server responding to a hardcoded route — verified via curl.
2. `Store` type in `internal/store`, map-backed, no HTTP knowledge, with `New`/`Get`/`Set`/`Delete`.
3. Routes wired: `GET/PUT/DELETE /store/{key}` using Go 1.22+ `ServeMux` method+path patterns and `r.PathValue`.
4. PUT reads the full raw request body via `io.ReadAll` and stores it as-is (no JSON parsing — deliberate simplification, discussed with user).
5. Refactored from inline closures to the `application`-struct + `handlers.go` method pattern (done in-editor by user, matches Let's Go convention; not yet committed — see above).

**Not yet done — this is the next task:**
- **Concurrency safety.** `store.go` currently has **no mutex**. A plain Go map is not safe for concurrent access; concurrent writes will panic. Next step is to have the user:
  - Learn `sync.Mutex` vs `sync.RWMutex` and choose (reads can be concurrent, writes must be exclusive — `RWMutex` is the natural fit, but let the user reason to that conclusion rather than being told outright).
  - Protect every map access in `store.go` with the appropriate lock (`Get`, `Set`, `Delete`).
  - Verify with `go run -race` while making concurrent requests (e.g. a small burst of concurrent curls or a simple concurrent test), confirming no race is reported.
- Once that's done, Stage 0 is complete and the "Resume line unlocked" is: *"Concurrent-safe in-memory key-value cache with an HTTP API in Go."*
- After that: commit, update the README with the *why* behind design decisions made so far (store/HTTP separation, status-code choices: 404 vs 200-empty, 201 vs 204, raw-body-as-value tradeoff), then move to Stage 1 (TTL + LRU).

## Open items / things to watch for
- README has not been created/updated yet with design-decision writeups — the guide treats this as important (interviewers read it), and it's fallen behind the code. Worth prompting the user on this.
- No tests exist yet for the store or handlers — also flagged by the guide as "write tests as you go," currently behind.
- The refactor commit (application struct / handlers.go) is still pending — don't let it get bundled silently into the next feature commit; keep it a separate, clean commit as originally planned.
