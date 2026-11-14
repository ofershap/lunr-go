# Security Policy

## Reporting a vulnerability

Please report security issues through GitHub's private vulnerability reporting
("Report a vulnerability" on the Security tab of this repository). Do not open
a public issue for a vulnerability.

## Scope notes

lunr-go indexes and searches caller-supplied documents and queries in
process: it performs no network calls. Fuzzy matching is bounded (edit
distance <= 3, term <= 32 UTF-16 code units) precisely because lunr.js lets
pathological values consume unbounded CPU; queries beyond the bounds return a
typed error. Bound document sizes in your own code when indexing untrusted
content.
