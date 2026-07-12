# Type Safety

## API Types

Use `request<T>()` with concrete response types. Feature service files should export the types used by pages rather than repeating inline shapes in components.

Reference files:
- `web/src/services/api.ts` defines `ApiError`, response parsing, pagination helpers, auth headers, and runtime API-response guards.
- `web/src/services/edgeNode.ts` defines `EdgeNode`, `HeartbeatRequest`, deployment types, and typed API methods.

## Backend Contract Shape

Backend JSON uses snake_case. Keep frontend service types aligned with that contract unless a mapper intentionally converts to a UI-specific shape. If a UI component needs camelCase props, do the conversion at the service or page boundary and keep both types explicit.

Do not assume optional backend fields are present. Types such as `HardwareInfo` in `edgeNode.ts` intentionally include optional fields for heartbeat data and legacy display fields.

## Runtime Guards

Use small runtime guards when parsing unknown JSON or protocol responses. `isApiResponseLike` and `isJsonResponse` in `web/src/services/api.ts` are local examples.

Avoid `as any` around API responses. If the backend returns a loose payload, define a narrow guard or a mapper that validates the fields the UI actually needs.

## Route And Menu Types

Routes depend on backend `Menu` values. Keep route config and sidebar route types in `web/src/router/types.ts` and use `String(menu.id || menu.code)` defensively as the generated router does.

When adding a menu-driven page, update:

- `menuComponentMap` in `web/src/router/index.tsx`
- Backend menu seed/migration if needed
- Locale key `menu:<code>` for every supported language
- Any typed route metadata in `web/src/router/`

## Common Mistakes

- Duplicating a backend response interface in several pages.
- Treating nullable/optional API fields as always available in JSX.
- Hiding failed JSON parsing behind `as T`.
- Using broad `Record<string, unknown>` props for domain components when a named interface would document the contract.
