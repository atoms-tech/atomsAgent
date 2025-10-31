# Spec: Chat History UI

## Summary
Expose chat history controls in the chat web UI so users can discover, browse, and reopen prior sessions.

## ADDED Requirements

### Requirement: Provide chat history entry point
- The chat interface MUST render a "History" trigger adjacent to existing header controls.
- Hovering over the trigger MUST reveal a combobox listing the five most recent sessions.
- The combobox MUST show each session title derived from the first user message (or "Untitled session" when unavailable).
- Selecting a session from the combobox MUST navigate to the dedicated history page with that session preselected.

#### Scenario: Hover displays combobox
1. GIVEN the chat page is loaded
2. WHEN the user hovers over the History trigger
3. THEN the UI shows a combobox overlay with recent session entries
4. AND the overlay dismisses when the pointer leaves the trigger or combobox region.

#### Scenario: Select session from combobox
1. GIVEN the combobox is open with at least one session
2. WHEN the user selects a session row
3. THEN the router navigates to `/chat/history` (or equivalent base path) with the session ID in the query params
4. AND the history page highlights the chosen session in the list.

### Requirement: Dedicated chat history page
- Clicking the History trigger (without selecting from the hover combobox) MUST navigate to a tabbed history page.
- The history page MUST contain at least a "Recent" tab and render a list of available sessions.
- Each session entry MUST display title, agent type, and last-updated timestamp when available.
- Opening a session from this page MUST return the user to the main chat view with that session loaded.

#### Scenario: History page navigation
1. GIVEN the user is on the chat page
2. WHEN the user clicks the History trigger
3. THEN the application navigates to the history page showing the Recent tab
4. AND an empty state message is shown if no sessions are available.

#### Scenario: Resume session from history page
1. GIVEN the history page lists at least one session
2. WHEN the user activates "Open" on a session
3. THEN the UI routes back to the chat conversation view
4. AND the conversation loads the prior messages for that session (or shows a toast if data unavailable).

### Requirement: Graceful fallback when history unavailable
- If the API returns an error or empty dataset, the UI MUST show an inline notice and keep the chat usable.
- Retry affordances (e.g., a "Reload" button) MUST be present to re-request history data.
- Instrumentation MUST log fetch failures to the console (and future telemetry hooks).

#### Scenario: API failure fallback
1. GIVEN the API stub responds with a non-2xx status
2. WHEN the history page attempts to fetch sessions
3. THEN the UI shows an error banner with retry option
4. AND selecting retry triggers another fetch without reloading the page.

