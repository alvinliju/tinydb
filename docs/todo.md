# TinyDB TODO

_Last updated: Saturday, July 19, 2025, 1:29 PM IST_

## Reminder
- **Single master only:** SPOF is accepted for maximal simplicity/learning. No master HA for now.
- **Everything else should stay stupidly minimal.**

---

## Core Features
- [ ] Implement write quorum logic (accept PUT/DELETE if 2/3 replicas succeed)
- [ ] Minimal error handling: fail fast on lost quorum, but only log replicas that fail behind
- [ ] Build a basic manual heal/repair tool to fix out-of-sync replicas (CLI script or endpoint)

## Observability & Ops
- [ ] `/status` and `/metrics` endpoints for master and volumes (basic: uptime, request count, healthy peers)
- [ ] Add simple logs with timestamps for requests, errors, and heal actions

## Configuration
- [ ] Parse YAML config for volumes (just ports and data dirs)
- [ ] Replace hardcoded CLI flags with YAML config for launching cluster

## DIY & Dev Experience
- [ ] Finalize Dockerfile(s) and a working `docker-compose.yml` for 1 master / 3 volumes
- [ ] Add a one-liner quickstart to README for spinning up the full cluster with Docker

## Testing & Docs
- [ ] Add minimal unit/integration tests for quorum and repair
- [ ] Write up REPO_STRUCTURE.md to show what folders/scripts are for (keep <1 page)
- [ ] Expand README just enough for usage, caveats, and “why single master” note

---

### What We’re **NOT** Doing (for now)
- No multi-master or master HA
- No fancy background repair, just a CLI/tool you run by hand
- No authentication or API-keys (see preferences) — all open by default
- No advanced CLI, no web UI — plain HTTP and basic shell scripts only

---

thanks for reading MF.
