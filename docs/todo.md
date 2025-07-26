# TinyDB TODO

_Last updated: July 25, 2025_

## Reminder
- **Single master only:** SPOF is accepted for maximal simplicity/learning. No master HA for now.
- **Everything else should stay stupidly minimal.**

---

## 🚨 CRITICAL FIXES (Do First!)
- [ ] **Fix DELETE crash bug** - volume server uses `log.Fatalf()` which crashes the entire server
- [ ] **Robust DELETE implementation** - all-or-nothing cleanup across ALL replicas (not quorum)
- [ ] **Better error handling** - replace `fmt.Println()` debug statements with proper logging

## Core Features  
- [x] ~~Implement write quorum logic (accept PUT/DELETE if 2/3 replicas succeed)~~ ✅ v0.2.0
- [ ] Implement proper DELETE logic - ensure ALL replicas cleaned up before removing from master
- [ ] Build a basic manual heal/repair tool to fix out-of-sync replicas (CLI script or endpoint)
- [ ] Add data consistency checker - background job to verify replica sync

## Production Readiness
- [ ] **Configuration management** - YAML config for cluster setup, limits, paths
- [ ] **Graceful shutdown** - handle SIGTERM/SIGINT properly, close DB connections cleanly  
- [ ] **Resource limits** - max file size, disk quota checking, concurrent request limits
- [ ] **Circuit breakers** - retry logic for volume operations with backoff
- [ ] **Request validation** - file size limits, malformed request handling

## Observability & Ops
- [ ] `/health` endpoint improvements - check DB connection, disk space, replica health
- [ ] **Prometheus metrics** - request counts, latencies, replica status, error rates
- [ ] **Structured logging** - replace debug prints with proper log levels and timestamps
- [ ] `/debug/pprof` endpoints for performance profiling (already have basic version)

## Configuration
- [ ] **YAML config file** - replace hardcoded volume groups, ports, limits
- [ ] **Environment variable overrides** - for Docker deployments
- [ ] **Config validation** - fail fast on startup with bad configs

## DIY & Dev Experience
- [ ] **Update Docker setup** - reflect new config file approach
- [ ] **Production docker-compose** - with proper resource limits, health checks
- [ ] **Load testing scripts** - standardized hey commands for benchmarking

## Testing & Docs
- [ ] **Delete operation tests** - verify all-replica cleanup behavior
- [ ] **Failure scenario tests** - partial replica failures, network timeouts
- [ ] **Production deployment guide** - config examples, monitoring setup
- [ ] Update README with v0.2.0 performance numbers and features

---

### What We're **NOT** Doing (for now)
- No multi-master or master HA  
- No authentication or API-keys — staying open by default for simplicity
- No fancy background repair, just a CLI tool you run manually
- No web UI — plain HTTP and shell scripts only
- No automatic data rebalancing — manual tools only

---

**Current Status:** v0.2.0 handles 1000+ req/sec but needs production hardening.  

thanks for reading MF.
