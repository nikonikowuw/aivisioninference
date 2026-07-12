# React Admin Frontend Specs

These specs cover `web/`, the React 19 + TypeScript + Vite admin SPA. The frontend owns admin pages, dynamic menu-driven routing, Chakra UI presentation, video/player UX, API service wrappers, i18n resources, and browser-side state.

## Guides

| Guide | Use When |
|-------|----------|
| [Directory Structure](./directory-structure.md) | Adding pages, services, shared components, locale files, routes, or assets |
| [Component Guidelines](./component-guidelines.md) | Building or changing Chakra UI components, admin pages, tables, forms, modals, or media panels |
| [Hook Guidelines](./hook-guidelines.md) | Adding shared stateful logic, polling, WebSocket, throttling, pagination, or date formatting |
| [State Management](./state-management.md) | Deciding where local, shared, server, route, auth, or media state belongs |
| [Type Safety](./type-safety.md) | Defining API types, DTO mappings, route/menu types, and runtime guards |
| [Quality Guidelines](./quality-guidelines.md) | Final checks for build, tests, accessibility, i18n, and performance |

## Local Anchors

- App entry: `web/src/index.tsx`, `web/src/App.tsx`
- Routes: `web/src/router/index.tsx`, `web/src/router/routes.config.ts`
- API boundary: `web/src/services/api.ts`, feature-specific files in `web/src/services/`
- Pages: `web/src/views/admin/`
- Shared UI: `web/src/components/`
- Hooks: `web/src/hooks/`
- Theme: `web/src/theme/`
- Locales: `web/src/locales/`

## Required Checks

- Type and production build: `cd web && npm run build`
- Unit/smoke tests: `cd web && npm run test`
- Local UI inspection when changing layout, media, or complex interaction: `cd web && npm run dev`
