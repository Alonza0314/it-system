# Frontend Dev README (Session Handoff)

## TL;DR (New Session 3-Min Start)
1. Workspace: `/home/alonza/it-system`
2. Frontend: `/home/alonza/it-system/frontend`
3. Install: `yarn install`
4. Run: `yarn dev`
5. Validate: `yarn build` and `yarn lint`
6. Core routes to smoke test:
   - `/login`
   - `/` (dashboard)
   - `/runner`
   - `/testcase`
   - `/tenant`
   - `/test`
   - `/test/task/:id`

## Critical Rules
- Use Yarn only. Never use npm/pnpm.
- `frontend/src/api` is generated (typescript-axios). OpenAPI changed -> regenerate first.
- Prefer minimal/local edits unless user asks for refactor.
- Reuse shared components (`Button`, `Modal`, `Switch`, `NotificationContainer`).

## Stack and Runtime
- React 19 + TypeScript + Vite + React Router + Axios
- API base URL:
  - `VITE_API_BASE_URL` if set
  - fallback: `${protocol}//${hostname}:8888`
- Local storage:
  - token key: `token`
  - username key: `username`
- Extra backend header from `getUserHeader()`:
  - `user: <username-from-localStorage-or-jwt>`

## App Structure

### Entry and Router
- Entry: `frontend/src/main.tsx`
- App router: `frontend/src/App.tsx`
- Auth guard: `RequireAuth` (token check)

### Main pages
- `frontend/src/page/login/LoginPage.tsx`
- `frontend/src/page/home/HomePage.tsx`
- `frontend/src/page/runner/RunnerPage.tsx`
- `frontend/src/page/test/TestPage.tsx`
- `frontend/src/page/test/TaskDetailPage.tsx`
- `frontend/src/page/testcase/TestcasePage.tsx`
- `frontend/src/page/tenant/TenantPage.tsx`

### Contexts
- `frontend/src/context/testcase-context.tsx`
- `frontend/src/context/tenant-context.tsx`

### API layer
- Generated API client: `frontend/src/api/*`
- Manual helper for runner APIs: `frontend/src/api/runner.ts`

## OpenAPI Regeneration Flow
Source spec: `/home/alonza/it-system/openapi.yaml`

When API contract changes:
1. Regenerate client into `frontend/src/api`
2. Fix compile errors (if response shape changed)
3. Run `yarn build`

Generator script:
- `/home/alonza/it-system/openapi-generator-docker.sh`

## Current UX/Behavior Contracts (Do Not Break)

### 1) Test Page (`frontend/src/page/test/TestPage.tsx`)
- `New Test` only toggles form open/close.
- Opening form does NOT bulk fetch all NF PRs.
- NF switch ON -> fetch that NF PR list only (`getGithubPRs(nf)`).
- Per-form cache exists; same NF should not re-fetch while form is open.
- Form close resets temporary states:
  - `prsByNf`
  - `loadingByNf`
  - `hasFetchedByNf`
  - `enabledNf`
  - `selectedPrByNf`
  - `selectedTestcases`
- Submit now has confirmation modal before actual API call.
- Submit modal must show selected testcases and NF/PR list.

### 2) Task Detail Page (`frontend/src/page/test/TaskDetailPage.tsx`)
- Cancel task action must use confirmation modal.
- Do not use browser native `confirm()`.

### 3) Runner Page (`frontend/src/page/runner/RunnerPage.tsx`)
- Poll runner list every 30 seconds while mounted.
- Runner cards are square grid layout (not full-row list).
- Delete runner must use shared `Modal` confirmation flow.
- Do not use browser native `confirm()`.

### 4) Dashboard (`frontend/src/page/home/HomePage.tsx`)
- Poll tasks + runners every 30 seconds while mounted.
- Includes KPI and detail summary for:
  - testcases
  - tenants
  - tasks (pending/ongoing)
  - runners (offline/idle/running)

## Visual Design System (Current Baseline)
Current style is: minimalist + modern + light theme + restrained colors.

### Tokens
- Global tokens in `frontend/src/index.css` (`--bg`, `--surface`, `--accent`, etc.)
- Prefer token usage over hardcoded colors for new UI.

### Tone
- Clean, premium, low-noise surfaces.
- Subtle accent color (blue family) with soft status tints.
- No heavy gradients, no random palette per page.

### Components
- Primary action -> shared `Button` (accent style)
- Confirm flows -> shared `Modal`
- Operation feedback -> `NotificationContainer`

## Error and Loading Pattern
- Parse backend error message from `error.response.data.message` when present.
- Fallback to deterministic message.
- Use toast notifications for success/error.
- Keep scoped loading flags (`isLoading`, `isSubmitting`, per-item loading).

## Build / Makefile Notes
- Root Makefile already orchestrates frontend build with Yarn.
- Frontend dist output: `frontend/dist`
- Root build sync target: `build/frontend`

## Fast Verification Checklist
1. `yarn install`
2. `yarn dev`
3. Login success
4. Dashboard data loads and 30s polling works
5. Runner delete shows modal and works
6. Test submit shows modal and works
7. Task detail cancel shows modal and works
8. `yarn build` and `yarn lint`

## Known Caveats
- `frontend/src/api` changes may be overwritten by regeneration.
- Notification IDs are timestamp-based; rapid operations may be close in time.

## Session Update Rules (for future AI sessions)
When continuing this frontend work:
1. Keep Yarn-only workflow.
2. Preserve behavior contracts above unless user explicitly asks to change them.
3. Keep style aligned to current token-based minimalist system.
4. Avoid introducing page-specific one-off visual systems.
5. Run at least error checks and ideally `yarn build` after meaningful changes.
