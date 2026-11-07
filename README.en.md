# vibeasik

> A standard layer for AI-generated code. Runs on your laptop or dev server and is called by coding agents over MCP.

[Bahasa Indonesia](README.md) is the primary language of this project; the docs in `docs/` are in Indonesian.
This page is a short English summary.

vibeasik is a local program for people who build software with AI agents. It keeps track of your projects, audits
AI-written code with **85 built-in rules**, manages ports and dev servers so projects do not collide, builds a code
map, and acts as an **MCP server** for Claude Code, Cursor, Antigravity and other agents.

**Everything runs locally.** Your code, project facts and API keys are never uploaded.

## Install

```bash
curl -fsSL https://vibeasik.com/install.sh | sh     # Linux / macOS
vibeasik serve          # start the engine on http://127.0.0.1:8765
vibeasik open           # open Studio (web UI), signed in automatically
vibeasik mcp install    # register with the coding agents installed on this machine
```

Windows: download the `.zip` from `https://vibeasik.com/dl/`.

## Editions

- **Community** (this repository, Apache-2.0): build it yourself with `make studio && make build`. 64 MCP tools.
- **Official** (installer / vibeasik.com downloads): the same core plus closed Pro modules that stay locked until you
  sign in with a Pro plan (`vibeasik login`). 123 MCP tools. Pro unlocks via an Ed25519-signed entitlement verified
  against a public key embedded in the binary.

## Knowledge Hub

`vibeasik knowledge pull` downloads signed knowledge packs (checksum + Ed25519 signature verified). Search always runs
on your machine. Only anonymous helpful/harmful counts and contributions you explicitly approve are ever sent.

## Security

The engine binds to `127.0.0.1`, requires a local token for every API call and checks the `Host` header. It can run
commands on your machine, so read [SECURITY.md](SECURITY.md) before exposing it to a network.

## Build from source

Go 1.26+, [bun](https://bun.sh), `make`:

```bash
git clone https://github.com/jendro/vibeasik.git && cd vibeasik
make hooks && make studio && make build && make test
```

Contributions are welcome: see [CONTRIBUTING.md](CONTRIBUTING.md). License: [Apache-2.0](LICENSE).
