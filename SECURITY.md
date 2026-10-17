# Security Policy

## Supported Versions

Security fixes are prioritized for the latest stable QuickDesk release and the
current default development branch. Older releases may not receive backports.
Self-hosted deployments should track the latest release and review security
fixes before exposing the Host, Signaling Server, WebClient, or TURN service to
an untrusted network.

## Reporting a Vulnerability

Do not disclose vulnerability details in a public issue, discussion, pull
request, or release comment.

Preferred route:

1. Open the repository's **Security** tab.
2. Select **Report a vulnerability**.
3. Submit the report through GitHub Private Vulnerability Reporting.

If GitHub Private Vulnerability Reporting is unavailable, contact the
maintainer at `870709864@qq.com` with the subject
`[QuickDesk Security]`. This is the public contact email currently listed on
<https://github.com/barry-ran>. Ask for an encrypted or private transfer channel
before sending especially sensitive exploit material.

Please include:

- the affected QuickDesk version or commit, platform, and component;
- the security impact and the conditions required to reproduce it;
- a minimal reproducer or proof of concept, if available;
- relevant logs or configuration, with unrelated credentials and personal data
  removed;
- whether you plan to publish the issue and how you would like to be credited.

Do not access data or systems you do not own, disrupt third-party deployments,
or test against public instances without explicit authorization.

## Scope

In scope:

- QuickDesk Host and Client, including input, file transfer, clipboard, update,
  trust, and skill-host boundaries;
- Signaling Server authentication, authorization, sessions, notifications,
  update delivery, storage, and deployment defaults;
- WebClient, Qt client, installers, and update verification;
- vulnerabilities caused by QuickDesk configuration or integration even when
  they originate in a third-party component.

Third-party issues should normally be reported to their upstream project as
well. Report them here when QuickDesk's packaging, defaults, or integration
makes the impact materially different.

Out of scope:

- findings that require access to a machine or account the tester does not own;
- social engineering, physical access, or denial of service against systems
  without authorization;
- reports that only identify an outdated release without a security impact;
- automated scanner output without a reproducible vulnerability or affected
  code path.

## Response and Coordinated Disclosure

Maintainers will make a best-effort attempt to acknowledge a private report,
validate the affected versions, prepare a fix, and coordinate disclosure.
Response and remediation times depend on severity, reproducibility, and
maintainer availability; no fixed SLA or bug bounty is promised.

Please allow a reasonable remediation period before publishing technical
details. Security advisories and release notes will describe affected versions
and mitigation or upgrade steps when appropriate.

## Maintainer Checklist

- Enable GitHub Private Vulnerability Reporting under **Settings > Security**
  so the preferred route above is available.
- Do not request exploit details in a public issue.
- Keep the private contact in this file current.
- Prepare an advisory, patched release, and upgrade guidance before public
  disclosure.
