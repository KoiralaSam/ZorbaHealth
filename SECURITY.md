# Security Policy

## Reporting a vulnerability

Please do not open public issues for suspected security vulnerabilities.

Until a dedicated security inbox is established, report vulnerabilities privately to the project maintainers through the repository owners' private contact channel and include:

- a description of the issue
- the affected components
- reproduction steps if available
- impact assessment
- any suggested mitigation

## Scope

Security reports are especially relevant for:

- authentication and authorization
- patient data handling
- PHI exposure through logs or APIs
- health-record access paths
- MCP tool access
- RabbitMQ event handling
- secrets management
- third-party provider integrations

## Response expectations

- acknowledgement as soon as practical
- triage and severity assessment
- coordinated remediation before public disclosure when appropriate

## Safe testing expectations

- do not access data you are not authorized to access
- do not attempt destructive tests against shared infrastructure
- use synthetic or local development data whenever possible

## Related security architecture notes

See `docs/security.md` for the repository's current technical security posture and planned hardening roadmap.
