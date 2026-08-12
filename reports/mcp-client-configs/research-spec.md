---
spec_version: "2.0"
research_topic: "MCP client configuration compatibility"
research_question: "How should a Go CLI manage one global MCP manifest across the requested macOS and Linux clients?"
topic_profile:
  type: "technology_tool"
  rationale: "The task concerns configuration schemas, paths, transports, and adapter boundaries."
depth_mode: "standard"
scope:
  time_range: "Current vendor documentation available on 2026-08-11"
  geography: "macOS and Linux"
  target_audience: "MCM implementers"
  as_of: "2026-08-11"
output:
  project_rules_ref: "none"
  directory: "reports/mcp-client-configs"
  report_slug: "mcp-client-configs"
facets:
  enabled:
    - primer
    - technology
    - landscape
  disabled:
    - name: market
      reason: "No market-sizing or adoption question."
    - name: investment
      reason: "No investment decision is requested."
    - name: risks
      reason: "Configuration risks are covered as technical compatibility limits."
    - name: outlook
      reason: "No forecast is required."
    - name: frontier
      reason: "No research-frontier question is required."
    - name: learning_path
      reason: "The audience needs an adapter decision, not a course."
artifact_plan:
  derivation: "standard depth plus primer, technology, and landscape facets"
  files:
    - README.md
    - data.csv
    - glossary.md
    - handbook/10-primer.md
    - handbook/20-technology.md
    - handbook/40-landscape.md
    - report.md
    - research-spec.md
    - sources.md
artifact_contracts:
  research_spec:
    format: "markdown"
  report:
    format: "markdown"
    sections:
      - "L0 summary"
      - "key conclusions"
      - "limits"
  handbook:
    chapters:
      - handbook/10-primer.md
      - handbook/20-technology.md
      - handbook/40-landscape.md
  glossary:
    format: "markdown"
  data:
    format: "csv"
    columns:
      - id
      - dimension
      - item
      - finding
      - value
      - unit
      - source_ids
      - confidence
  charts:
    types: []
  raw: {}
  examples: {}
  dimensions:
    items: []
  sources:
    format: "markdown"
source_policy:
  preferred:
    - "Official vendor documentation"
    - "First-party source repositories"
  required_verification:
    - "At least two independent sources for cross-client conclusions"
    - "Single-vendor facts are marked medium confidence unless independently corroborated"
  prohibited:
    - "Unaudited third-party setup guides as proof of a client contract"
validation_rules:
  min_sources_per_key_fact: 2
  conflict_resolution: "Prefer the user's explicit project constraint for MCM output targets; retain vendor-documentation conflicts as runtime verification requirements."
  facet_required_questions_policy: "answer_or_open_or_not_applicable"
  cross_document_consistency: true
investment_policy:
  enabled: false
  mode: "decision_support_only"
  prohibit_personalized_advice: true
  prohibit_unsourced_price_targets: true
  require_scenarios: true
wave_plan:
  wave_1_scanners:
    - "Vendor documentation for Cursor, VS Code, Claude Code, Codex, Qoder, and OpenCode"
    - "First-party repositories for mcpc and philschmid/mcp-cli"
  wave_2_deep_questions:
    - "Compare target schema roots, user-level locations, transport support, and credential references."
execution:
  max_items_per_swarm: 128
  total_budget: "Single-agent standard research; eight products and nine adapters."
  max_rounds: "2"
execution_audit:
  batches:
    - selected_engine: "serial official-source web research"
      fallback_reason: "No AgentSwarm tool is available in this environment."
      owner_scope: "primary agent"
      provenance: "sources.md entries 1-13"
knowledge_injection:
  documents:
    - "User-confirmed constraints: Go CLI; macOS and Linux; global MCM home ~/.mcm; Qoder CLI and IDE are separate targets; legacy SSE is unsupported."
---

# Research Spec: MCP Client Configuration Compatibility

## 1. Goal

Give MCM implementers a verified compatibility map for the requested clients,
so one manifest can be transformed into native configuration without claiming
that all client schemas or transports are identical.

## 2. Questions

1. What user-level file or invocation should each adapter target?
2. Which configuration root and fields does each client expect?
3. Which current transports can be emitted safely for each target, excluding
   legacy SSE?

## 3. Scope

Included products: mcpc, philschmid/mcp-cli, Qoder CLI, Qoder IDE, Cursor,
Codex, Claude Code, VS Code, and OpenCode.

Excluded: Windows paths, project-level synchronization, client installation,
credential entry, and applying any configuration on this machine.

## 4. Required answers

| facet | status | result |
|---|---|---|
| primer | answered_with_sources | handbook/10-primer.md |
| technology | answered_with_sources | handbook/20-technology.md |
| landscape | answered_with_sources | handbook/40-landscape.md |

## 5. Validation

The required bundle files exist, report.md has frontmatter, data rows cite
sources, and all single-source client facts are explicitly confidence-limited.
