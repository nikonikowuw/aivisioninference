# Frontend Quality Guidelines

## Required Local Rules

Read the relevant `.rules/` documents before frontend changes:

- `.rules/frontend-design.md` for visual/layout/i18n conventions.
- `.rules/ui-ux-pro-max.md` for accessibility and operational admin UX.
- `.rules/vercel-react-best-practices.md` for React/Vite performance and render correctness.

## i18n

All display text belongs in `web/src/locales/`. Include buttons, table headers, input hint text, toast text, modal copy, tooltips, empty states, chart labels, player messages, and error text.

Supported locales are configured under `web/src/i18n/` and currently include `zh-CN`, `zh-TW`, `en-US`, `id-ID`, `ja-JP`, and `ko-KR`.

## Build And Tests

Run:

```bash
cd web && npm run build
cd web && npm run test
```

Add or update tests when changing:

- Route/menu generation (`web/src/router/index.test.tsx`)
- API response parsing and auth behavior
- Video player setup/cleanup (`web/src/components/VideoPlayer/VideoPlayer.smoke.test.tsx`)
- Complex forms, permission logic, or high-risk list filtering

## Performance

Keep heavy dependencies out of the shared initial path. Video libraries, charts, Markdown, cropping, file preview, and similar large modules should be route-level or component-level lazy imports. The FLV path in `VideoPlayer` imports `flv.js` only when needed.

Memoize expensive table columns, chart options, and media configs. Clean up WebSocket, timers, media players, event listeners, and AbortControllers.

## Accessibility And Responsive Checks

Verify:

- Keyboard focus is visible.
- Icon-only controls have `aria-label`.
- Form errors are connected to fields.
- Destructive actions require confirmation.
- Tables have loading/empty/error/pagination states.
- Mobile layouts do not create accidental horizontal overflow unless the table is intentionally scrollable.
- Dark mode keeps text, chart, border, and tooltip contrast readable.

## Anti-Patterns

- Direct `fetch` calls from components.
- Hardcoded UI strings in JSX.
- Local style systems that bypass Chakra theme.
- Global state libraries for local page state.
- Rendering more than about 50 operational rows without pagination or virtualization.
