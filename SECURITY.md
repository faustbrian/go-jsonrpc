# Security policy

## Supported versions

Security fixes target the latest stable major release. Maintainers may also
patch an older major when impact and adoption justify it. Unsupported versions
may receive an advisory without a backport.

| Version | Supported |
| --- | --- |
| 1.x | Yes |
| < 1.0 | No |
| Unreleased `main` | Best effort |

## Report a vulnerability privately

Use the repository's **Security** tab to submit a private vulnerability report
through GitHub Security Advisories. If that feature is unavailable, contact a
maintainer privately through their verified GitHub profile and ask for a secure
reporting channel. Do not include exploit details in a public issue or pull
request.

Include the affected version, impact, reproduction steps, relevant payloads,
and any proposed mitigation. Remove credentials, personal data, and unrelated
production information.

## Response process

Maintainers aim to acknowledge a complete report within three business days,
provide an initial severity assessment within seven business days, and share a
remediation plan after reproducing the issue. Timelines may change with
complexity, but the reporter will receive status updates during investigation.

Validated vulnerabilities are fixed on a private branch when possible. The
release includes tests, an advisory, affected versions, severity, and upgrade
or mitigation instructions. Disclosure is coordinated with the reporter after
a fixed release is available, unless active exploitation requires earlier
notice.

## Security boundaries

The package parses untrusted JSON and crosses application error boundaries.
Particularly sensitive areas include body-size enforcement, panic containment,
error-data disclosure, ambiguous protocol members, malformed UTF-8, batch
amplification, custom transports, header handling, and context cancellation.
Authentication, authorization, TLS configuration, rate limiting, and
application validation remain the adopter's responsibility.
