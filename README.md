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

Swapping that for a disk-backed server such as `http.Dir("static")` works locally, where the real `static/` folder sits next to the binary, and fails in the container, where it was never copied. The result is a site that serves HTML with every stylesheet and image returning 404: readable text, no styling.

If the site ever renders as unstyled text again, check `/static/style.css` first:

```sh
curl -s -o /dev/null -w '%{http_code}\n' https://cdavenport.io/static/style.css
```

`200` means assets are fine and the problem is elsewhere. `404` means the embed or the `/static/` route is missing. Verify against the container, not just `go run .`:

```sh
docker build -t cdavenport.io .
docker run --rm -p 8080:8080 cdavenport.io
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/static/style.css
```
