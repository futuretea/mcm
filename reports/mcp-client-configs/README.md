# MCP Client Configuration Research

- Profile: technology_tool
- Depth: standard
- As of: 2026-08-11
- Scope: macOS and Linux user-level configuration for the requested clients

## Contents

- report.md: decision summary and implementation direction.
- data.csv: structured compatibility facts and confidence levels.
- sources.md: primary sources and the user-confirmed Qoder IDE path.
- handbook/10-primer.md: configuration concepts and boundaries.
- handbook/20-technology.md: adapter model and target matrix.
- handbook/40-landscape.md: product differences and rollout order.
- glossary.md: terms used consistently across the bundle.

## Reading path

1. Read report.md for the initial product boundary.
2. Use handbook/20-technology.md when implementing adapters.
3. Check data.csv and sources.md before updating a client adapter.

## Confidence

Cross-client conclusions have two or more independent primary sources. A client
specific path or schema normally has one authoritative vendor source and is
therefore marked medium confidence. The Qoder IDE path is a user-confirmed
project constraint; current public IDE documentation confirms its schema but
not that filesystem location, so it remains low confidence until runtime
verification.

Investment analysis: not included. Disabled facets: market, investment, risks,
outlook, frontier, and learning_path; this is an implementation-compatibility
study rather than a market or learning report.
