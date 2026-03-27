# ctx-ignore

> Scan your repo. Generate `.claudeignore`, `.cursorignore`, and `.contextignore`. Keep your AI coding tools focused on what matters.

```bash
brew install arniesaha/tap/ctx-ignore
ctx-ignore scan ./my-repo
```

---

## The problem

Your AI coding assistant sees everything in your repo — `node_modules`, lockfiles, generated code, build artifacts, test fixtures. That's thousands of tokens of noise on every request, slowing responses and burning context window.

Most tools solve this at query time (fetching only what's needed). `ctx-ignore` solves it at setup time: run it once, and your tools stay clean permanently.

## How it works

```
ctx-ignore scan ./my-repo
```

1. Walks the repo and scores every file by LLM usefulness
2. Flags noise: generated files, binaries, lockfiles, build artifacts, boilerplate
3. Shows you a ranked list with explanations
4. Generates ignore files your AI tools actually read

**Output files:**

| File | Used by |
|------|---------|
| `.contextignore` | Source of truth — annotated, human-readable |
| `.claudeignore` | Claude Code |
| `.cursorignore` | Cursor |

Optionally patches `.gitignore` too.

## Commands

```bash
ctx-ignore scan [path]     # Scan repo, show ranked noise report + generate ignore files
ctx-ignore stats [path]    # Show token cost breakdown by file
ctx-ignore diff [path]     # Compare token usage before vs after .contextignore
ctx-ignore init            # Generate ignore files from sensible defaults (no scan)
```

## Example output

```
ctx-ignore scan .

Scanning 847 files...

NOISE (recommended to ignore):
  🔴 package-lock.json        — 42,310 tokens  lockfile, auto-generated
  🔴 node_modules/            — 2.1M tokens    dependencies, never edit
  🔴 dist/                    — 18,400 tokens  build output, generated
  🔴 coverage/                — 9,200 tokens   test artifacts
  🟡 *.test.ts (34 files)     — 28,100 tokens  test files (optional to ignore)
  🟡 migrations/ (12 files)   — 4,300 tokens   DB migrations, rarely relevant

SIGNAL (keep in context):
  ✅ src/                     — 38,200 tokens
  ✅ README.md                — 1,100 tokens
  ✅ go.mod                   — 340 tokens

Before: 2.4M tokens  →  After: 39,640 tokens  (98.4% reduction)

Generated:
  ✅ .contextignore
  ✅ .claudeignore
  ✅ .cursorignore
```

## Install

```bash
# macOS / Linux (Homebrew)
brew install arniesaha/tap/ctx-ignore

# Linux (curl)
curl -sSf https://install.ctx-ignore.dev | sh

# Go
go install github.com/arniesaha/ctx-ignore@latest
```

## Philosophy

- **Setup time, not query time.** Fix the problem once, don't patch it on every request.
- **Explain the why.** Every ignored pattern has a reason — the `.contextignore` file is human-readable and editable.
- **Zero dependencies.** Single binary, no config required, works on any repo.
- **Tool-agnostic.** Claude Code, Cursor, Copilot — one source of truth, multiple output formats.

## Status

🚧 v0.1 in development

---

MIT License
