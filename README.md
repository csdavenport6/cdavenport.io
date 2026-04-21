# cdavenport.io

The Go blog that serves https://cdavenport.io. Reads Markdown posts with YAML frontmatter, renders them with Go `html/template` and `goldmark`, and embeds templates and static assets via `//go:embed`.

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

Push to `main`. CI builds, tags, and pushes an image to `ghcr.io/csdavenport6/cdavenport.io`, then signs a POST to `https://deploy.cdavenport.io/hooks/blog` to trigger a production redeploy. Secrets required in GitHub Actions:

- `DEPLOY_HOOK_URL`: e.g. `https://deploy.cdavenport.io/hooks/blog`
- `DEPLOY_HOOK_SECRET`: HMAC secret shared with the server's `webhook.env`.

The deploy-webhook receiver lives in [csdavenport6/infra](https://github.com/csdavenport6/infra); see its [webhook runbook](https://github.com/csdavenport6/infra/blob/main/docs/webhook-runbook.md) for architecture, troubleshooting, and secret rotation.

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

## Rotating the deploy hook secret

1. Generate a new secret: `openssl rand -hex 32`.
2. Update the value in 1Password (`infra BLOG_HOOK_SECRET`).
3. Update the repo secret `DEPLOY_HOOK_SECRET`.
4. Update `/etc/infra/webhook.env` on the droplet (`BLOG_HOOK_SECRET=<new>`), then `docker compose up -d webhook` in `~/infra`.
