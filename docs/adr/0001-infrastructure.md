# ADR-0001: Infrastructure Foundations

**Status:** Accepted
**Date:** Project start

---

## SQLite as Database

**Decision:** Use SQLite as the primary database via GORM with auto-migration. No migration files.

**Rationale:** Zero-ops for MVP; single-user or small-team usage; easy file-based backup. No infrastructure to provision or maintain.

**Alternatives rejected:**
- Postgres — operational overhead not justified at this scale.

---

## No Repository Abstraction Layer

**Decision:** All domain packages access `storage.DB` directly. No repository interfaces or structs between handlers and GORM.

**Rationale:** Reduces boilerplate for a small codebase. Direct `storage.DB` access is acceptable at MVP scale. Repository pattern can be introduced incrementally if packages grow beyond current size.

**Alternatives rejected:**
- Repository pattern — adds indirection without current benefit.

---

## Monorepo (api + ui in one repo)

**Decision:** Keep `pathfinder-api/` and `pathfinder-ui/` in the same Git repository, orchestrated by a single `Taskfile.yml`.

**Rationale:** Simplifies local dev and CI; single `Taskfile.yml` orchestrates both services; reduces coordination overhead for a one-person team.

**Alternatives rejected:**
- Separate repos — unnecessary coordination cost at this team size.
