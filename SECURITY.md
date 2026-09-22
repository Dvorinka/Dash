# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| latest release | yes |
| older releases | no |

Only the latest release receives fixes.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting:
**Security → Advisories → Report a vulnerability** on this repository.

Do not open a public issue for security problems.

Please include:

- affected version/commit
- steps to reproduce or a proof of concept
- impact assessment

You will get an acknowledgement within a few days. If the report is accepted,
a fix ships in the next release and you are credited in the advisory (unless
you prefer otherwise).

## Scope notes

Dash is designed for trusted private networks — it ships **without
authentication** by design. Exposure to the public internet is a deployment
choice outside the project's control, but issues that remain exploitable on a
LAN (SSRF in checkers, injection, token leakage, unsafe schemes) are in scope.
