# ADR 0001: Keep a Runnable Default as the Template Source

- Status: Accepted
- Date: 2026-08-06

## Context

The repository has two audiences. Contributors need a complete application they
can run, test, secure, and reason about. Project generation needs versioned
source material from which a Selected Stack can be produced. Maintaining a
separate example application and template would let their behavior drift.

## Decision

The Template Source remains a complete Runnable Default using Fiber, MySQL, and
the database-backed queue. The manifest identifies the supported selection
space and template version. A Generated Project contains only its Selected
Stack and becomes independent after generation.

Template releases are assembled only from Git-tracked files with deterministic
archive metadata and a checksum. The manifest supplies the default release
version; an explicit release version may override it for controlled packaging.

Optional integrations are represented by narrow readiness or wiring seams in
the Template Source. Their concrete adapters are selected or added during
generation without pulling framework or driver concerns into the core.

## Consequences

- The default repository must remain runnable and production-shaped.
- Changes to shared source must make sense both before and after generation.
- Generated combinations need separate compatibility coverage as that matrix
  grows.
- Untracked local files, ignored secrets, logs, and build output cannot enter a
  template release.
- The Runnable Default demonstrates one choice and does not make that choice
  mandatory for Generated Projects.
