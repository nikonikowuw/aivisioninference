# State Management

## State Locations

Use the smallest state owner that fits the workflow:

- Component local state for forms, modals, loading/error flags, selected rows, and one-page filters.
- URL/route state for navigable identifiers and pages that should survive refresh/back navigation.
- Context for app-wide concerns such as auth/session and shared layout settings.
- Service modules for API shape and request functions, not for mutable UI state.
- Refs for mutable player/socket/timer/canvas handles that should not cause renders.

The app does not use Redux, Zustand, MobX, or React Query. Do not add a global state library for a local page workflow.

## Server State

Server data is fetched through `web/src/services/api.ts` and feature APIs. Represent request state explicitly with `loading`, `error`, and data state. Lists should preserve page/page_size/filters and ask the backend for the correct slice.

Reference files:
- `web/src/services/api.ts`
- `web/src/services/edgeNode.ts`
- `web/src/hooks/usePagination.ts`
- `web/src/hooks/useFilter.ts`

## Auth And Errors

Access tokens are stored via `getAccessToken`, `setAccessToken`, and `clearAccessToken` in `web/src/services/api.ts`. Unauthorized responses redirect to `/auth/sign-in` from the central request layer.

Do not duplicate token parsing, `Authorization` header setup, or unauthorized redirect behavior in feature components.

## Media State

Media playback state belongs near the player instance. `VideoPlayer` tracks current protocol, loading, error, fallback info, cleanup, and mounted status locally because each player has independent stream lifecycle.

Do not put player instances, WebRTC/FLV/HLS handles, or canvas loops in global state.

## Derived State

If a value can be computed from props and state during render, compute it directly or with `useMemo`. Do not mirror it into state through an effect unless the user can edit it independently.

Avoid state synchronization loops such as: API data -> local copy -> effect writes another copy -> render reads the second copy.
