# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes       |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report security issues via [GitHub's private security advisory system](https://github.com/shapestone/flow-wire-diagram/security/advisories/new).

You can expect an acknowledgement within 48 hours and a resolution or status update within 14 days.

## Scope

`wire-fix` reads and rewrites Markdown files on the local filesystem. It has no network access, no authentication, and handles no credentials or sensitive data. The primary security concern is malformed input causing unexpected file writes.
