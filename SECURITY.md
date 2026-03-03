# Security Policy

## Philosophy

BenchTalks is designed so that a compromised server reveals as little as possible. The server cannot decrypt messages, cannot identify users, and holds no persistent data. A server breach exposes: the number of active rooms, the number of connected clients, and admin token hashes (SHA-256, one per room, useless without the original token).

That said, vulnerabilities in the application code, the WebSocket handling, or the federation layer could still affect users. Please report them.

## Supported versions

Only the latest release on the main branch is actively maintained.

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report security issues by emailing: **b3ncht4lks@protonmail.com**

Include:
- A description of the vulnerability
- Steps to reproduce
- What an attacker could do by exploiting it
- Your assessment of severity

You will receive an acknowledgement as soon as possible. If the issue is confirmed, a fix will be prioritised and you will be credited in the release notes unless you prefer otherwise.

## What counts as a vulnerability

- Any way for the server to recover or observe message content
- Any way to impersonate a room admin without the admin token
- Any way to persist data on the server that should be ephemeral
- Denial of service vulnerabilities that can take down a bench with low effort
- Vulnerabilities in the NATS federation that allow cross-bench traffic interception
- XSS or other client-side vulnerabilities in the room interface

## What does not count

- The server operator reading their own server logs — operators are trusted for their own bench by design
- The fact that room IDs are not secret — security comes from the encryption key, not the room ID
- Loss of messages when a server restarts — **this is intentional**
