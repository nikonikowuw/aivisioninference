# Hook Guidelines

## Existing Hook Roles

- `web/src/hooks/usePagination.ts`: list pagination state.
- `web/src/hooks/useFilter.ts`: filter/search state.
- `web/src/hooks/useDateFormat.ts`: date/time formatting.
- `web/src/hooks/useWebSocket.ts`: WebSocket lifecycle.
- `web/src/hooks/useThrottledInference.ts`: video/canvas inference overlay throttling.

Check these before writing a new hook.

## Lifecycle Ownership

Hooks that subscribe to browser APIs, WebSocket, timers, video players, canvas loops, or event listeners must clean up in their effect return. Store mutable instance handles in `useRef`, not state, when changes should not trigger rendering.

Reference files:
- `web/src/components/VideoPlayer/index.tsx` stores media cleanup in refs and tears down on URL/protocol changes.
- `web/src/hooks/useWebSocket.ts` owns socket lifecycle rather than scattering it across pages.

## Data Fetching

This project does not use React Query. Keep data fetching in components or custom hooks with explicit loading/error state, and call typed service wrappers from `web/src/services/`.

Independent requests should run with `Promise.all` or `Promise.allSettled` rather than sequential awaits. Use `AbortController` or mounted guards for effects where late responses can update an unmounted component.

## Naming And Return Shape

Use `useX` names and return stable, narrow values:

- Current state needed by the caller.
- Setter/command functions wrapped in `useCallback` when passed down.
- Refs only when the caller must attach them to DOM/media elements.

Avoid hooks that hide business workflows behind ambiguous names such as `useData` or return large untyped objects.

## Performance

Use `useMemo` for large chart options, TanStack columns, derived list transforms, and media configuration objects. Use lazy `useState(() => initialValue)` for expensive initial values. Use `useDeferredValue` or `useTransition` for search/filter interactions that can block rendering.
