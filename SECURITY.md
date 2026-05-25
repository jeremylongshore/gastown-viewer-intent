# Security policy

## Supported versions

| Version | Supported |
|---|---|
| 0.5.x   | ✓ |
| 0.4.x   | ✓ |
| Earlier | — |

## Reporting a vulnerability

If you discover a security vulnerability, report it by:

1. **Do NOT** open a public issue.
2. Email the maintainers directly (see `CODEOWNERS` or the repo
   description).
3. Include:

   - Description of the vulnerability.
   - Steps to reproduce.
   - Potential impact.
   - Suggested fix (if any).

We will respond within 48 hours and work with you to address the issue.

## Security considerations

For the full threat model, see `THREAT_MODEL.md` in the repo root. Key
points:

- The daemon shells out to the `bd` and `gt` CLIs — ensure both binaries
  are from a trusted source.
- The daemon binds loopback only by default; `--host=0.0.0.0` is
  refused at startup.
- An Origin allowlist middleware rejects cross-origin requests, defending
  against DNS rebinding and CSRF from other tabs on the dev box.
- A 256-bit session token is generated on every daemon start and
  persisted to `~/.config/gvid/token` (mode `0600`). State-mutating
  endpoints (none ship today) require the token.
- The memories panel is read-only by architectural invariant. Class A
  (partner names) and Class B (secret-pattern) matches in memory content
  are redacted server-side per the
  [memory classification policy](000-docs/005-PP-POLICY-memories-classification-2026-05-24.md).

For deployments outside a personal dev box:

- Do NOT expose the daemon beyond `localhost`. The threat model assumes
  a single trusted user.
- Run behind a reverse proxy with its own auth if multi-user access is
  ever required (out of scope for this product today).
- Keep dependencies updated; the `.github/workflows/doc-quality.yml`
  and `ci.yaml` workflows include hash-pinned policy files via the
  `@intentsolutions/audit-harness` tooling.
