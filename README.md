# cdavenport.io

The Go blog that serves https://cdavenport.io. Reads Markdown posts with YAML frontmatter and renders them with Go `html/template` and `goldmark`.

Static assets (`static/`) are compiled into the binary with `//go:embed`. Templates (`templates/`) and posts (`posts/`) are **not** embedded; they are read from disk at startup. See [How files are served](#how-files-are-served).

## Run locally

```sh
go run .
```

Open http://127.0.0.1:8080/.

## Run with Docker

```sh
docker build -t cdavenport.io .
docker run --rm -p 8080:8080 cdavenport.io
```

Or with the dev compose file:

```sh
docker compose -f compose.dev.yml up --build
```

## Deploy

Push to `main`. CI runs `go test ./...`, then builds and pushes an image to `ghcr.io/csdavenport6/cdavenport.io`, tagged `:latest` and `:sha-<short-sha>`.

Deployment is **pull-based**: the homelab's Komodo watches ghcr.io for a new `:latest` and redeploys the `blog` stack on its own poll interval. CI does not push to the server, so there are no deploy secrets and no inbound webhook. A push to `main` goes live once CI finishes and Komodo next polls.

Homelab configuration lives in [csdavenport6/infra](https://github.com/csdavenport6/infra).

> The old Digital Ocean droplet and signed-webhook deploy (`DEPLOY_HOOK_URL` / `DEPLOY_HOOK_SECRET`) was removed during the homelab migration. Those secrets are no longer read by CI and can be deleted from the repo settings.

### Cloudflare sits in front of the origin

Requests do not reach the homelab directly. Cloudflare proxies the domain, and it caches static assets aggressively:

| Path | `cf-cache-status` | Cached? |
| --- | --- | --- |
| `/static/...` | `HIT` | Yes, `cache-control: max-age=14400` (4 hours) |
| `/`, `/posts/...` | `DYNAMIC` | No, passed through to the origin |

**After a deploy that changes anything under `static/`, purge the Cloudflare cache.** Otherwise the edge keeps serving the previous copy for up to 4 hours, to you and to every visitor.

This matters more than it sounds, because *broken* responses are cached too. When the `/static/` route was missing, requests for `style.css` returned the homepage HTML with `200 OK`, and Cloudflare cached that wrong response for 4 hours under the stylesheet's URL. Redeploying the fix did not visibly change anything: the origin was correct and the edge was still handing out stale HTML.

A deploy that "did not work" is very often a deploy that worked, hidden behind this cache. Confirm at the origin before concluding the deploy failed.

## Writing posts

Drop a new `.md` file into `posts/` with frontmatter:

```markdown
---
title: "Post Title"
date: 2026-04-19
slug: "post-title"
tags: ["go"]
---

Post body in Markdown.
```

## How files are served

The three asset directories reach production by two different routes. This matters when changing how files are served, because a mistake here can pass locally and still break production.

| Directory | How it reaches production | Served by |
| --- | --- | --- |
| `static/` | Compiled into the binary by `//go:embed static` in `main.go` | `/static/` route |
| `templates/` | `COPY templates/ /data/templates/` in the Dockerfile | Read from disk by `NewServer` |
| `posts/` | `COPY posts/ /data/posts/` in the Dockerfile | Read from disk by `LoadPosts` |

The Dockerfile deliberately does **not** copy `static/` into the runtime image, because the embed already put those files inside the binary.

The consequence: `/static/` must be served from the embedded filesystem, never from disk.

```go
//go:embed static
var staticFiles embed.FS

staticFS, _ := fs.Sub(staticFiles, "static")
mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
```

Swapping that for a disk-backed server such as `http.Dir("static")` works locally, where the real `static/` folder sits next to the binary, and fails in the container, where it was never copied.

### Diagnosing unstyled text

Check the **Content-Type**, not the status code, and always **bust the cache**:

```sh
curl -s -o /dev/null -w '%{content_type}\n' "https://cdavenport.io/static/style.css?bust=$(date +%s)"
```

- `text/css; charset=utf-8` means assets are fine and the problem is elsewhere (see [Still looks broken](#still-looks-broken-after-a-correct-deploy)).
- `text/html; charset=utf-8` means the `/static/` route is missing.

Two separate traps make this failure easy to misdiagnose.

**1. A missing route returns 200, not 404.** `mux.HandleFunc("/", srv.HandleIndex)` registers `/` as a catch-all prefix in Go's `ServeMux`, so any path without a more specific route falls through to it. With `/static/` unregistered, a request for `style.css` is handled by `HandleIndex` and returns the homepage with `200 OK`. The browser tries to parse HTML as a stylesheet, silently discards it, and renders unstyled. Status code alone will tell you everything is fine. It is not.

**2. Without a cache buster you are testing Cloudflare, not the origin.** A plain `curl` of a `/static/` URL is answered by the edge, which may hold a 4-hour-old copy. Polling that URL after a deploy can report the old state indefinitely while the origin is already correct. Add `?bust=$(date +%s)`, or read `cf-cache-status` to see which layer answered:

```sh
curl -sI "https://cdavenport.io/static/style.css" | grep -iE '^(cf-cache-status|age|content-type|cache-control):'
```

Verify against the container, not just `go run .`, since a disk-backed regression passes locally:

```sh
docker build -t cdavenport.io .
docker run --rm -p 8080:8080 cdavenport.io
curl -s -o /dev/null -w '%{content_type}\n' http://127.0.0.1:8080/static/style.css
```

### Still looks broken after a correct deploy

If the origin serves `text/css` but the site still renders unstyled, the stale copy is in a cache, not the code. Work outward:

1. **Your browser.** Hard-refresh (`Cmd+Shift+R`) or open a private window. A normal reload will not help, because the browser considers its copy fresh for the full `max-age`.
2. **Cloudflare.** Purge the cache from the dashboard so other visitors are not stuck on the old copy for up to 4 hours.

Confirm the fix reached production by comparing bytes rather than trusting appearance:

```sh
curl -s "https://cdavenport.io/static/style.css?bust=$(date +%s)" | shasum -a 256
shasum -a 256 static/style.css
```

Matching hashes mean production is serving exactly what is in the repo, and anything still wrong is downstream of delivery.
