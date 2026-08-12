# Glossary

| term | definition | allowed_synonyms | avoid | first_used_in |
|---|---|---|---|---|
| adapter | A client-specific renderer, reader, and writer for one native configuration contract. | client adapter | universal writer | handbook/20-technology.md |
| canonical manifest | MCM's own source-of-truth file under ~/.mcm. | source manifest | client config | report.md |
| stdio | A local MCP server launched as a subprocess and connected by standard input/output. | local process | shell transport | handbook/10-primer.md |
| Streamable HTTP | The current HTTP MCP transport, normally represented as http in client configuration. | HTTP MCP | generic web API | handbook/20-technology.md |
| SSE | The legacy Server-Sent Events MCP transport. | Server-Sent Events | HTTP fallback | handbook/20-technology.md |
| user scope | Configuration available across projects for one operating-system account. | global scope | project scope | handbook/10-primer.md |
| direct adapter | An adapter that writes the native user-level client configuration. | native writer | export adapter | handbook/20-technology.md |
| export adapter | An adapter that writes a portable configuration file for a client to consume explicitly. | file export | direct adapter | handbook/20-technology.md |
