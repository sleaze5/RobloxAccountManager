# Design

This file is the authoritative specification for visual design, component styling, interaction states, accessibility behavior, and motion. When another document conflicts with it on these topics, follow this file.

The interface is quiet, precise, technical, dense, and fast. It is a near-black, almost grayscale UI with thin borders, small brightness steps, consistent geometry, and very short transitions.

## Sources of truth

- `frontend/src/styles/tokens.css` defines every color, font, size, spacing, radius, shell dimension, duration, and easing value. Use its variables. Do not write raw values that a token covers, and do not redefine tokens in component styles.
- `frontend/src/styles/base.css` holds global element styles and the reduced-motion guard. Each feature has its own stylesheet in `frontend/src/styles/`, imported by `index.css`.
- `frontend/src/lib/shared/presence.ts` defines the enter and exit transitions for menus, dialogs, tooltips, and notifications. Reuse them.
- [`assets/icon.svg`](../assets/icon.svg) is the editable source of the application icon.
- Use Phosphor icons when an appropriate icon exists. Import each component directly, for example `import GearIcon from "phosphor-svelte/lib/GearIcon"`. Do not use barrel imports or an icon registry.
- Use the default `regular` icon weight, `light` for title-bar controls, `bold` for compact checkbox marks, and `fill` for filled states such as favorites. Use the `weight` prop instead of SVG `fill` or `strokeWidth` overrides.
- Add `aria-hidden="true"` to decorative icons. Keep accessible names on icon-only controls and meaningful standalone icons.

Reuse shared styles and components. Keep component-specific styles local only when they cannot be shared. Keep similar components visually and behaviorally consistent.

## Embedded browser UI

The user must never see UI that looks or behaves like an exposed embedded browser. Everything the frontend shows must look intentionally designed as part of the application.

- After the frontend is ready, keep every user-facing interaction inside the WebView UI. Native operating-system UI is acceptable only during startup, before the frontend can show its own UI.
- Do not use `alert()`, `confirm()`, or `prompt()`. Use the application's dialogs and notifications.
- Do not use the `title` attribute for tooltips. Use `data-tooltip` (and optionally `data-tooltip-side`, and `data-tooltip-strong` to bold one part of the text), which `lib/shared/Tooltip.svelte` renders.
- Add `novalidate` to every `<form>`. Show validation errors in the application UI, not in browser validation popups.
- The default context menu is disabled for the whole application. Show a context menu only where a custom one is implemented, such as the account list.
- Standard HTML controls are allowed only when they are styled to match the design. Replace a control that cannot be fully restyled. Use `lib/shared/Select.svelte` instead of `<select>`, and the `.cookie-selection` checkbox markup instead of a native checkbox.
- If no suitable component exists, create one that follows this file.

## Layout and surfaces

- Design for a 900px by 500px window at 100% display scaling. This is also the minimum window size.
- Use `--color-bg` for the main content plane, `--color-surface` for persistent chrome (navigation, sidebars, tool rails), and `--color-base` for the darkest layer.
- Use `--color-dialog` for dialogs and `--color-overlay` for menus, popovers, and other floating surfaces. Use `--color-surface-raised` for compact opaque controls.
- Separate regions with a background step plus a 1px border, not with large shadows. Use `--color-border` by default and `--color-border-strong` for focused controls, selected controls, and floating surfaces that need extra separation. Avoid bright outlines.
- Keep the shell flat. Use the shell geometry tokens for the title bar, headers, status strip, and sidebars.

## Typography

- Use Geist (`--font-sans`) for interface text and Geist Mono (`--font-mono`) for technical text such as paths, values, IDs, and measurements.
- Bundle both fonts as local, normal-style WOFF2 variable fonts from Vercel's pinned release, with weights 100 through 900. Preload Geist in `frontend/index.html`. Do not fetch fonts at runtime. Keep the pinned upstream sources in `THIRD_PARTY_NOTICES.md` and the font license in `THIRD_PARTY_LICENSES.txt`.
- Keep text between 10px and 16px. Use 13px for body text, 12px for compact labels and helper text, 15px semibold for page headings, and 10px to 12px monospace for technical text.
- Use weight 400 for body text, 500 for selected navigation, compact emphasis, and control labels, and 600 for headings and strong labels. Use 700 rarely.
- Create hierarchy with weight, color, spacing, and grouping, not oversized text.
- Use `--color-text` for body text and important labels, `--color-text-muted` for timestamps, helper text, metadata, and secondary labels, and `--color-text-faint` for placeholders, inactive navigation, file paths, disabled labels, and tertiary information. Use `--color-text-bright` sparingly for dominant actions and the strongest emphasis. Do not use arbitrary opacity for text.

## Spacing and geometry

- Use the 4px spacing scale. Use 4px for tight internal spacing, 8px for compact controls, 12px for row and group gaps, and 16px for larger internal padding. Keep page and surface padding between 12px and 20px, and use 16px to 24px between major content groups.
- Prefer `gap` over margins for rows and columns.
- Use 24px controls for compact icon actions, 28px controls for ordinary buttons and menu rows, and 32px controls for text fields and dominant actions.
- Use the radius scale: chips and tags 4px, controls 6px, cards and panels 8px, menus, popovers, and dialogs 10px, and large surfaces 12px. Do not use other radius values, and do not turn every control into a pill.
- Keep nested curves concentric: child radius = outer radius - inset.

## Controls and states

- Style inputs as shallow plates: `--color-input` background, 1px border, control radius, and faint placeholders. On focus, use the stronger border.
- Use the solid high-contrast button (`--color-solid` with `--color-on-solid`) only when one action must dominate. Keep secondary buttons quiet: transparent background, muted text, and a 1px border.
- Keep hover and active feedback low contrast (`--color-hover`, `--color-active`). A selected state is a slightly stronger version of the resting surface. Do not use large brightness changes, and do not replace hover states with shadows.
- Every interactive element exposes the hover, focus, disabled, loading, and error states that apply to it.
- Preserve keyboard access and keep visible focus indicators.
- Do not rely on color alone. Pair semantic color with text, an icon, or another cue.
- Keep the normal interface grayscale and the accent (`--color-accent`) neutral. Use the success, warning, danger, and presence colors only to communicate meaning. Avoid large decorative status fills.
- Style inline code with the monospace font and the neutral `--color-fill-subtle` background. Do not color inline code like a link.

## Motion

Motion must feel immediate. If an animation feels like something the user waits for, it is too slow.

Use these durations and easings:

- Hover color: `70ms` to `90ms`, `--ease-standard`
- Press feedback: `70ms`, `--ease-standard`
- Tooltip: `90ms`, `--ease-sharp`
- Menu or popover enter: `110ms`, `--ease-sharp`
- Menu or popover exit: `80ms`, `--ease-standard`
- Dialog enter: `130ms`, `--ease-sharp`
- Dialog exit: `80ms`, `--ease-standard`
- Small panel enter: `140ms` maximum, `--ease-sharp`

- No normal transition exceeds `160ms`.
- Animate movement only with `transform` and `opacity`, over small distances (`--motion-distance`, `--motion-scale`). Do not use bounce, spring overshoot, slow fades, or decorative loops.
- Direct hover and focus feedback may also transition `color`, `background-color`, and `border-color`.
- Do not animate layout properties, `filter`, `backdrop-filter`, blur, or `box-shadow`. Never use `transition: all`. List each transitioned property.
- The only continuous animation is a functional busy spinner. Do not run an idle `requestAnimationFrame` loop. Apply `will-change` only to an element that is about to animate.
- Keep menus, dialogs, tooltips, notifications, and operation errors mounted during their exit transition. Reopening a closing surface reverses or cancels the exit. It does not queue another animation. Restore focus only after a dialog finishes closing.
- When the selected account, settings category, or filtered settings result changes, use a restrained entrance for the new content. Do not animate the outgoing content. Do not stagger account rows or tag rows.
- Validation feedback may replay one horizontal nudge of at most 2px for 140ms. Cancel the previous nudge before replaying it. Do not hide or delay the error text.

## Reduced motion

The Animations setting in Settings → User Interface chooses System (the default), Reduced, or Full. System follows the operating system's `prefers-reduced-motion` preference.

`lib/shared/motion.ts` resolves the setting and sets the `data-reduced-motion` attribute on the root element. `base.css` defines the reduced-motion guard for that attribute, and `presence.ts` reads it through `reducedMotion()`. Define the guard only once, and do not query `prefers-reduced-motion` anywhere else. Under reduced motion:

- Exits complete immediately.
- The spinner stops but stays visible as a static busy indicator.
- Selected, expanded, loading, success, warning, and error states stay understandable without animation.
