# Component Guidelines

## Design System

Use Chakra UI and the project theme. New admin UI should reuse existing layout rhythm, semantic tokens, and shared components before adding custom style systems. Do not introduce another component library for ordinary forms, tables, modals, cards, or buttons.

Reference files:
- `web/src/theme/`
- `web/src/components/search-bar/`
- `web/src/components/pagination/`
- `web/src/components/confirm-dialog/`
- `web/src/views/admin/system/SystemConfig.tsx`

## Admin Page Shape

Admin pages should provide the states users need for repeated operations:

- Loading state
- Empty state
- Error state
- Pagination for large lists
- Clear filters/search controls
- Confirmation for destructive actions
- Toast or inline feedback for async operations

Tables should use server pagination/filtering when the backend supports it. Avoid fetching all rows and filtering in the browser for operational lists.

## Props And Composition

Define component props as TypeScript interfaces near the component unless the type is shared across modules. Keep prop names domain-specific and avoid passing untyped `Record<string, any>` through component trees.

Lift child components to module scope when they do not need closure-local state. Do not declare subcomponents inside a render function if it causes remounts and state loss.

## Chakra And Accessibility

Use Chakra primitives for form and interaction semantics:

- `FormControl`, `FormLabel`, `Input`, `Select`, `Textarea`, `FormErrorMessage`
- `Modal`, `AlertDialog`, `Toast`, `Badge`, `Tooltip`, `Spinner`, `Skeleton`

Icon-only buttons need `aria-label`. Form fields need visible or accessible labels. Do not rely on input hint text as the only label.

## Media Components

Video components must clean up player instances, subscriptions, timers, and canvas work on unmount. `web/src/components/VideoPlayer/index.tsx` uses refs for video/canvas/player cleanup and lazy-loads `flv.js` only for FLV playback.

Keep media states explicit: connecting, playing, fallback protocol, failed protocols, offline, and no stream. The existing `VideoPlayer` shows protocol fallback and loading/error overlays; follow that style for new media surfaces.

## Styling Rules

Prefer Chakra responsive props and `useColorModeValue` over local CSS constants. If a chart/player needs fixed colors, centralize those constants near the chart/player adapter and name their purpose.

Do not hardcode UI text in JSX. Use `react-i18next`, including toasts, modal copy, table headers, empty states, tooltips, chart labels, and player errors.
