# Proposal: Introduce Chat History Access

## Summary
Design and implement a chat history management experience so users can review and reopen previous agent chats via the chat UI. The work should cover front-end scaffolding, API surface expectations, and graceful fallbacks while backend persistence is wired in by agentapi.

## Motivation
Operators need to resume prior sessions without losing context. Today the chat UI only shows the live conversation for the current session. A structured history flow will let users browse past interactions and rehydrate conversations.

## Desired Outcomes
- A discoverable entry point in the chat UI for managing history (hover reveals combobox; click navigates to a chat history view).
- Tabbed history page capable of listing previous sessions, previewing metadata, and opening a session.
- API and service stubs in agentapi to back the UI, returning sensible fallbacks until real data is wired.
- Documented expectations for storage (e.g., Redis vs Supabase) to support future implementation.

## Non-Goals
- Implementing the actual persistence layer for chat transcripts.
- Migrating existing data stores.
- Finalizing production-ready UX polish beyond basic navigation and layout.

## Open Questions
- What data source will ultimately store session metadata (Redis vs Supabase vs other)?
- Should histories be scoped per-agent, per-organization, or global?
- Are there retention or privacy constraints that impact the design?

