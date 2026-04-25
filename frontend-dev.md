# Frontend Development Memory (for AI handoff)

## Purpose
This document is the single source of truth for frontend development handoff.
Goal: when switching machine or chat session, AI can continue without re-learning architecture and style.

## Project Scope
- Workspace root: /home/alonza/it-system
- Frontend root: /home/alonza/it-system/frontend
- Stack: React 19 + TypeScript + Vite + React Router + Axios
- Package manager: Yarn only

## Package Manager Rule (Critical)
Use Yarn for all frontend tasks.
Do not use npm.
Do not use pnpm.

Recommended commands:
- Install deps: yarn install
- Dev server: yarn dev
- Build: yarn build
- Lint: yarn lint
- Preview build: yarn preview

## Build and Integration Notes
- Root Makefile already uses Yarn for frontend build.
- Frontend module Makefile delegates to root Makefile.
- Frontend build output goes to frontend/dist and copied to build/frontend by root Makefile.

## Environment and Runtime
- API base path is resolved in page/context layer with:
  - VITE_API_BASE_URL if provided
  - fallback: protocol + hostname + :8888
- Auth token is stored in localStorage key: token
- Username is stored in localStorage key: username
- Additional backend header:
  - key: user
  - value: username resolved from localStorage or JWT claims

## Routing and App Shell
- Entry: frontend/src/main.tsx
- Router root: frontend/src/App.tsx
- Protected routes use RequireAuth (token check only).
- Main layout with sidebar navigation:
  - Dashboard
  - Testcase
  - Test
  - Tenant

## Directory Responsibilities
- frontend/src/page
  - login: auth page and sign-in flow
  - layout: app shell and sidebar
  - home: dashboard cards and refresh
  - testcase: testcase CRUD page
  - tenant: tenant CRUD page with permission fallback
  - test: test creation panel with NF switch and PR selection
- frontend/src/context
  - testcase-context.tsx: testcase state, loading, refresh/add/delete
  - tenant-context.tsx: tenant state, loading, permission handling, refresh/add/delete
- frontend/src/components
  - button, modal, switch
  - notifications + errorBox + successBox
  - test/NfPrSelector
  - stats/stats-card (available component)
- frontend/src/api
  - generated OpenAPI client (typescript-axios)
  - do not manually edit generated files unless intentional temporary patch

## OpenAPI Generator Workflow
Source of API contract is openapi.yaml in workspace root.
Generated client path is fixed:
- output: frontend/src/api
- generator command script: openapi-generator-docker.sh

Generator command behavior:
- Uses Docker image openapitools/openapi-generator-cli
- Input spec: /local/openapi.yaml
- Generator: typescript-axios
- Output: /local/frontend/src/api

After OpenAPI changes:
1. regenerate frontend api client
2. verify compile with yarn build
3. adjust page/context usage if response shapes changed

## API Usage Pattern in Frontend
Current project pattern is direct page/context invocation of generated DefaultApi.
No separate service layer currently.

Common pattern:
- create DefaultApi with Configuration(basePath, accessToken callback)
- call generated method directly in page or context
- pass extra headers using getUserHeader when needed
- parse backend message from error.response.data.message

Examples by responsibility:
- login page directly calls api.login
- testcase and tenant pages use context methods
- test page directly calls api.getGithubPRs for per-NF PR list

## Test Page Behavior Contract (Important)
File: frontend/src/page/test/TestPage.tsx

Current required behavior:
- New Test button only toggles form open/close.
- When form opens:
  - no bulk GitHub request is sent immediately
- Each NF switch ON triggers request for that NF only.
- Request API:
  - getGithubPRs(nf)
  - response uses data.prs
- In-form cache behavior:
  - fetched PR list is cached per NF in local state
  - same NF should not re-request while form remains open
  - toggling switch OFF then ON again should reuse cached data (no duplicate request)
- Cache reset behavior:
  - when form closes, clear per-form temporary state:
    - prsByNf
    - loadingByNf
    - hasFetchedByNf
    - enabledNf
    - selectedPrByNf
    - selectedTestcases

NF mapping contract:
- apiName must match backend/OpenAPI enum exactly.
- UPF apiName is upf (not go-upf).

## UI and Visual Style Baseline (Do Not Drift)
This codebase currently uses a soft, light dashboard style.
Keep future UI consistent with these rules.

### Typography and overall tone
- Sans-serif UI with Segoe UI / Noto Sans fallback.
- Medium to bold headings, clear hierarchy, readable spacing.
- Tone: professional, clean, slightly modern glass effect on some surfaces.

### Color language
- Base background: light slate/gray gradients.
- Core text: deep slate (#0f172a family).
- Accent system mainly blue/cyan tones for structure and focus.
- Success and error toast colors are green/red semantic.

Important existing mixed accent:
- Shared primary Button and active Switch use indigo-purple gradient.
- Page-level action buttons are often solid blue.
Preserve this current behavior unless doing a full design refactor.
Do not introduce new random accent palettes page-by-page.

### Surfaces and shapes
- Rounded corners around 8 to 20px.
- Cards/tables use white or translucent white surface.
- Borders are subtle slate/blue tints.
- Shadows are soft and non-heavy.

### Motion and interaction
- Subtle transitions only (hover lift, fade/slide, panel open animation).
- Notifications slide in from right.
- Form panel in Test page uses expand/collapse animation.

### Layout behavior
- Sidebar + content two-column desktop shell.
- Mobile/tablet: stack to single column at medium breakpoints.
- Tables remain readable with horizontal overflow wrapper.

### Component consistency rules
- Reuse shared Button, Modal, Switch components.
- Reuse NotificationContainer for operation feedback.
- For forms, use existing input border, radius, and focus ring pattern.
- For page headers, keep heading + action alignment pattern.

## Error Handling and UX Pattern
- Extract backend message from response.data.message when available.
- Fallback to deterministic message string if unavailable.
- Show result via notification boxes, not alert().
- Keep loading flags explicit and scoped (global page vs per-item).

## TypeScript and Lint Rules
- TS strict mode enabled.
- noUnusedLocals and noUnusedParameters enabled.
- React hooks linting enabled.
- Prefer explicit interfaces for payload and context contracts.

## AI Implementation Rules for Future Sessions
When AI edits frontend in this repo, follow these rules:

1. Keep Yarn-only commands.
2. Prefer minimal, local changes; avoid broad refactors unless requested.
3. Respect direct generated API usage pattern already used by pages/contexts.
4. If API schema changed, regenerate frontend/src/api first, then fix compile errors.
5. Keep visual style aligned with current light-slate + blue system and existing component patterns.
6. Do not invent a new design system or dark theme unless explicitly requested.
7. Preserve responsive behavior in each page-level CSS module.
8. Preserve notification-based feedback flow.

## Fast Onboarding Checklist (for new machine/session)
1. Confirm in frontend/package.json that packageManager is Yarn.
2. Run yarn install in frontend.
3. Verify backend URL strategy via VITE_API_BASE_URL or default :8888 fallback.
4. If API changed, regenerate client from openapi.yaml into frontend/src/api.
5. Run yarn build and yarn lint.
6. Start with yarn dev and validate login + dashboard + testcase + tenant + test page basic flow.

## Known Current Caveats
- frontend/src/api is generated output; local patches can be overwritten on regeneration.
- Notification IDs are timestamp-based; very rapid operations may still be close in time.
- Some unused demo styles/components may remain for future expansion (for example stats card).

## Last Verified Snapshot
- Architecture scanned from current workspace frontend source.
- Includes current Test page per-NF lazy PR loading and per-form cache reset behavior.
