# Design

This file is the authoritative specification for visual design, component styling, interaction states, accessibility behavior, UI text, and motion. When another document conflicts with it on these topics, follow this file.

The interface is quiet, precise, technical, dense, and fast. It is a near-black, almost grayscale UI with thin borders, small brightness steps, consistent geometry, and very short transitions.

When this file does not cover a case, copy the closest existing pattern in the application. If no pattern fits, follow the description above and name the new pattern in the change summary.

## Sources of truth

- `frontend/src/styles/tokens.css` defines every color, font, size, spacing, radius, opacity, layer, shell dimension, duration, and easing value. Use its variables. Do not write raw values that a token covers, and do not redefine tokens in component styles.
- Add a token for a value that repeats or that defines the design. A one-off value that no token covers may stay raw.
- When TypeScript needs a token value, read it from the computed style. Do not repeat token values in TypeScript.
- `frontend/src/styles/base.css` holds global element styles and the reduced-motion guard. Each feature has its own stylesheet in `frontend/src/styles/`, imported by `index.css`.
- `frontend/src/lib/shared/presence.ts` defines the enter and exit transitions for menus, dialogs, tooltips, and notifications. Reuse them.
- [`assets/icon.svg`](../assets/icon.svg) is the editable source of the application icon.

## Building blocks

Build from these parts before you write new styles. If a part lacks a variant you need, extend the part. Do not create a parallel style for the same role. Keep component-specific styles local only when they cannot be shared.

Buttons:

- `.primary-action`: the one dominant action in a dialog, drawer, or empty state. The Launch button uses this style.
- `.control-button`: a bordered secondary button. Add `.text-button` when the button has a text label. Use it without `.text-button` for a bordered icon-only button.
- `.icon-action`: a borderless icon-only button.
- `.danger-button` and `.warning-button`: confirmation actions with a danger or warning tone. See [Destructive actions](#destructive-actions).

Inputs:

- `.search-field`: a search input with a leading icon and a clear button.
- `.modal-field`: a labeled text field.
- `lib/shared/Select.svelte`: a dropdown. Use it instead of `<select>`.
- `lib/shared/NumberStepper.svelte`: a numeric input.
- `.cookie-selection`: a checkbox. Use it instead of a native checkbox.
- `.settings-toggle`: a switch. Its input has `role="switch"`.
- `.settings-radio-option`: a radio button.
- `.segmented-control`: two to four mutually exclusive options that stay visible.

Surfaces and feedback:

- `.account-menu`: menus and context menus.
- `.modal-backdrop`, `.modal-card`, `.modal-heading`, and `.modal-actions`: dialogs.
- `.modal-warning`, `.modal-error`, and `.modal-note`: inline notices.
- `lib/shared/OperationError.svelte`: a failed operation that has no specific place on the page.
- `lib/notifications/notification-center.svelte.ts`: notifications.
- `lib/shared/Tooltip.svelte`: tooltips, through the `data-tooltip` attribute.

Content:

- `.empty-state`: empty, loading, and unavailable states for a page or pane.
- `.tag-pill`: tags.
- `.profile-facts`: a list of labels and values, written as a `<dl>`.
- `.text-block`: multi-line technical text.
- `lib/shared/Timestamp.svelte`, `lib/shared/CompactNumber.svelte`, and `lib/shared/SensitiveValue.svelte`: formatted dates, compact numbers, and hidden secrets.

Layout:

- `.workbench`: the sidebar and detail pane layout, with `.account-explorer` as the sidebar, `lib/layout/SidebarResizer.svelte` as the resizer, and `.status-bar` as the status strip.

Name a new shared class by its role, such as `.list-row`, not by the feature that first uses it. Several existing shared classes have feature names, such as `.account-menu` and `.cookie-selection`. Reuse them anyway. Do not copy them under a new name.

## Embedded browser UI

The user must never see UI that looks or behaves like an exposed embedded browser. Everything the frontend shows must look intentionally designed as part of the application.

- After the frontend is ready, keep every user-facing interaction inside the WebView UI. Native operating-system UI is acceptable only during startup, before the frontend can show its own UI.
- Do not use `alert()`, `confirm()`, or `prompt()`. Use the application's dialogs and notifications.
- Do not use the `title` attribute for tooltips. Use `data-tooltip` (and optionally `data-tooltip-side`, and `data-tooltip-strong` to bold one part of the text), which `lib/shared/Tooltip.svelte` renders.
- Add `novalidate` to every `<form>`. Show validation errors in the application UI, not in browser validation popups.
- The default context menu is disabled for the whole application. Show a context menu only where a custom one is implemented, such as the account list.
- Standard HTML controls are allowed only when they are styled to match the design. Replace a control that cannot be fully restyled.
- If no suitable building block exists, create one that follows this file.

## Layout and surfaces

- Design for a 900px by 500px window at 100% display scaling. This is also the minimum window size. Check every layout change at this size with the launch panel open at its default height.
- At the minimum size, the detail pane next to a default-width sidebar is about 680px wide. Make tables and fixed-column grids fit that width. Hide or truncate lower-priority columns before you add horizontal scrolling.
- Use `--color-bg` for the main content plane, `--color-surface` for persistent chrome (navigation, sidebars, tool rails), and `--color-base` for the darkest layer.
- Use `--color-dialog` for dialogs and `--color-overlay` for menus, popovers, and other floating surfaces. Use `--color-surface-raised` for compact opaque controls.
- Separate regions with a background step plus a 1px border, not with large shadows. Use `--color-border` by default and `--color-border-strong` for selected controls and floating surfaces that need extra separation. Avoid bright outlines.
- Keep the shell flat. Use the shell geometry tokens for the title bar, headers, status strip, and sidebars.
- Show each value once per screen. For example, do not show the same count in a sidebar header and in the status strip.

Accounts, Games, and Logs Explorer share one page anatomy:

1. The page tabs sit below the title bar.
2. A resizable, collapsible sidebar on `--color-surface` holds a header row (title, count, and icon actions), a search row, and a list.
3. A detail pane on `--color-bg` shows the selected item, or an `.empty-state` when nothing is selected.
4. A status strip at the bottom shows context for the current selection.

Settings replaces the page tabs with a header that has a back button.

Stack floating layers in this order, from lowest to highest. Define each layer as a token in `tokens.css`.

1. Page content
2. Menus and popovers anchored to a control
3. Operation errors
4. Context menus
5. Notifications
6. Dialogs
7. The vault lock screen

Menus, dropdowns, and tooltips open in the browser's top layer, above every layer in this list, so that a scroll area or the launch panel never cuts them off. Position a menu with `placeMenuAtAnchor`, `placeMenuAtPoint`, or the `floatMenu` action from `presence.ts`, which lift it automatically. Use `liftToTopLayer` for a floating surface that positions itself. Do not use `position: fixed` alone for a menu inside a scroll area, because `container-type` on an ancestor clips it.

## Typography

- Use Geist (`--font-sans`) for interface text and Geist Mono (`--font-mono`) for technical text such as paths, values, IDs, and measurements.
- Bundle both fonts as local, normal-style WOFF2 variable fonts from Vercel's pinned release, with weights 100 through 900. Preload Geist in `frontend/index.html`. Do not fetch fonts at runtime. Keep the pinned upstream sources in `THIRD_PARTY_NOTICES.md` and the font license in `THIRD_PARTY_LICENSES.txt`.
- Keep text between 10px and 16px.
- Use 13px for body text and list-row titles, and 12px for controls, compact labels, and helper text. Use 10px to 12px monospace for technical text.
- Use 15px semibold for page titles, pane titles, and dialog titles. Use 13px semibold `--color-text` for subheadings: sub-page titles, card titles, chat thread titles, and empty-state titles. Use 12px semibold `--color-text-muted` for section headings inside a page or pane. Use 16px semibold only for the name of the selected account or game in its detail header. Do not use other heading sizes.
- Use weight 400 for body text. Use 500 for buttons, menu items, control labels, selected navigation, and compact emphasis. Use 600 for headings, names, and strong labels. Use 700 only for numeric count badges.
- Create hierarchy with weight, color, spacing, and grouping, not oversized text.

Text colors:

- Use `--color-text` for body text, names, and important labels.
- Use `--color-text-muted` for timestamps, helper text, metadata, and secondary labels.
- Use `--color-text-faint` for placeholders, inactive navigation, file paths, and tertiary information. Use it only on `--color-base`, `--color-bg`, `--color-surface`, `--color-dialog`, and `--color-overlay`. On raised, hover, and selected fills, use `--color-text-muted` or a stronger color.
- Use `--color-text-bright` sparingly for primary buttons and the strongest emphasis.
- Do not use accent tokens or arbitrary opacity for text.

## Icons

- Use Phosphor icons when an appropriate icon exists. Import each component directly, for example `import GearIcon from "phosphor-svelte/lib/GearIcon"`. Do not use barrel imports or an icon registry.
- Use the default `regular` weight, `light` for title-bar controls, `bold` for compact checkbox marks, and `fill` for filled states such as favorites. Use the `weight` prop instead of SVG `fill` or `strokeWidth` overrides.
- Use 16px icons by default in buttons, menu items, list rows, and fields.
- Use 14px icons in 24px controls, in tags, and next to 12px text. Use 12px icons next to 11px or smaller text. Use 11px icons for checkbox marks.
- Use 18px icons for title-bar controls and 28px icons for empty states.
- Use one icon for one meaning. Do not reuse a page's icon for an unrelated action, and do not use a trash icon for an action that only clears input.
- Add `aria-hidden="true"` to decorative icons. Keep accessible names on icon-only controls and meaningful standalone icons.

## Spacing and geometry

- Use the 4px spacing scale. Use 4px for tight internal spacing, 8px for compact controls, 12px for row and group gaps, and 16px for larger internal padding. Keep page and surface padding between 12px and 20px, and use 16px to 24px between major content groups.
- Prefer `gap` over margins for rows and columns.
- Use 24px controls for compact icon actions, 28px controls for ordinary buttons and menu rows, and 32px controls for text fields and dominant actions.
- Keep nested curves concentric: child radius = outer radius - inset.

Use the radius scale:

- 4px (`--radius-chip`): chips, tags, badges, checkboxes, and segmented-control items
- 6px (`--radius-control`): buttons, inputs, menu items, inline notices, avatars, and tooltips
- 8px (`--radius-panel`): cards, panels, and text blocks
- 10px (`--radius-surface`): menus, popovers, dialogs, and notifications
- 12px (`--radius-large`): large surfaces

Use `50%` for status dots, radio buttons, and switch thumbs. Use a full pill only for the switch track. Do not use other radius values, and do not turn other controls into pills.

## Controls and states

Inputs:

- Style inputs as shallow plates: `--color-input` background, 1px `--color-border` border, control radius, and faint placeholders. On focus, use `--color-border-focus`.
- Give every text field a visible label. With a visible label, the field border may stay subtle.

Buttons:

- Style the primary button with a `--color-surface-raised` fill, a `--color-border-strong` border, and `--color-text-bright` text. On hover, use `--color-surface-raised-hover` and `--color-border-focus`. Use at most one primary button in a dialog, drawer, or empty state.
- Keep secondary buttons quiet: transparent background, `--color-border` border, and `--color-text-muted` text. On hover, use `--color-hover` and `--color-text`.
- Reserve `--color-solid` and `--color-on-solid` for checked checkboxes, count badges, and text selection. Do not use them for buttons.

Hover and selection:

- Keep hover and active feedback low contrast (`--color-hover`, `--color-active`). Do not use large brightness changes, and do not replace hover states with shadows.
- Mark the selected row in a list with two cues: a `--indicator-size` bar in `--color-accent` on the left edge, and a `--color-active` fill. The bar carries the selection. Keep the fill quieter than `--color-hover`, because a brighter fill makes the list heavy.
- When the pointer is over a selected row, show the `--color-hover` fill and keep the bar.
- Draw the selection bar the same way in every list. The bar must not shift the row content.
- Mark the selected page tab with a `--indicator-size` bar in `--color-accent` on its bottom edge.
- Mark the selected item in a sidebar navigation list with a `--color-active` fill, a `--color-border` border, and medium weight.

Disabled and other states:

- Every interactive element exposes the hover, focus, disabled, loading, and error states that apply to it.
- Show a disabled control with `--opacity-disabled` and `cursor: not-allowed`. Do not also change its colors. When the reason is not obvious, explain it in visible text near the control.

Color:

- Keep the normal interface grayscale. Use `--color-accent` and `--color-accent-strong` only for indicators and badges, such as the selection bar, the tab bar, and filter counts.
- Use the success, warning, danger, and presence colors only to communicate meaning. Avoid large decorative status fills.
- Do not color an action by what it does, such as green for Join. Use semantic color on an action only when the action is destructive or needs a warning.
- Style inline code with the monospace font and the neutral `--color-fill-subtle` background. Do not color inline code like a link.

## Patterns

### Lists

- Give list rows the shared `.list-row` class, `--list-row-height`, and a 1px `--color-border-subtle` separator. Add the `selected` class to the selected row. `.list-row` draws the hover and selection treatment from [Controls and states](#controls-and-states).
- When a list can be reordered, support dragging and `Ctrl+Shift+ArrowUp` and `Ctrl+Shift+ArrowDown`. Announce the new position through a polite live region.
- Distinguish an empty list ("No accounts yet") from a list with no matches ("No matching accounts"). Tell the user how to change each state.

### Menus

- Open a menu from its trigger on click. Open a context menu on right-click, `Shift+F10`, and the `ContextMenu` key.
- Move between menu items with the arrow keys, `Home`, and `End`. `Escape` closes the menu and returns focus to the trigger.
- Do not make right-click the only way to reach an important action. Also offer the action from a visible control, such as a More actions button.
- Make menu items 28px tall with 12px medium text, a 16px icon on the left, and a check on the right for a toggled item.
- Put destructive items last, after a separator, with `--color-danger` text.

### Dialogs

1. Render the dialog as a `.modal-card` inside a `.modal-backdrop`. Use `modalFocus` and the dialog transitions from `presence.ts`. `modalFocus` moves focus to the first text field, or to the first control when the dialog has no text field.
2. Use `role="dialog"`. Use `role="alertdialog"` for a confirmation.
3. Put the title and an optional one-sentence description at the top.
4. Put a 24px close button in the top-right corner. Omit the close button only when the user must complete the dialog, such as the vault lock screen.
5. Show errors inside the dialog with `.modal-error`, above the actions.
6. Put the actions at the bottom right: Cancel first, then the main action.
7. Name the main action with a verb and an object, such as "Remove account". Do not use "OK" or "Yes".
8. While an action runs, disable the dialog's controls and show the progress form of the label, such as "Removing…".
9. Make `Escape` and the close button behave like Cancel. While an action runs, they do nothing.

### Destructive actions

- Confirm every destructive action in a dialog that names the affected item and the consequence.
- Use `.danger-button` only for actions that delete data or end something that cannot resume. Use the primary style for a reversible action, such as locking the vault.
- State when an action cannot be undone.
- When an action can delete data that has no backup, require the user to type a confirmation word, such as `RESET`.

### Forms and validation

- Put a label above each field in 12px `--color-text-muted`. Add "(optional)" to the label of an optional field.
- Validate a field after it loses focus or when the user submits the form. Do not show errors for a field the user has not changed.
- Show a field error below the field in `--color-danger-muted`. Set `aria-invalid="true"` on the field and give it a `--color-danger-border` border.
- Show the items of a requirement checklist in `--color-text-muted` until the user types. After that, mark each requirement as met or unmet with an icon and color.

### Loading, empty, and error states

- Show loading with a spinner and short text, such as "Loading accounts…". Set `aria-busy` on the region that is loading.
- When content refreshes, keep the existing content visible and replace the refresh icon with the spinner.
- Build an empty state from a 28px icon, a title that says what is missing, one sentence that says how to fill it, and an action button when one exists.
- Show an error next to the thing that failed, in a `role="alert"` element. Use `OperationError` when the failure has no specific place. Use a notification for the result of background work.

### Tooltips

- Write tooltip text in sentence case without a final period. Keep it short.
- On an icon-only control, make the tooltip match or expand the accessible name.
- Do not put information only in a tooltip, because screen readers do not read tooltip text. Put required information in the accessible name, a description, or visible text.
- Close the tooltip on `Escape`.

### Notifications

- Show notifications only through the notification center.
- Give each notification a short title and one sentence. Use a past-tense title for a result, such as "Accounts saved".
- Give an error notification an action that helps the user recover when one exists.

## Accessibility

- Give all text a contrast ratio of at least 4.5:1 against its background.
- Give meaningful icons, state indicators, focus rings, and the boundaries of checkboxes, radio buttons, and switches a contrast ratio of at least 3:1 against their background.
- Do not rely on color alone. Pair semantic color with text, an icon, or another cue.
- Make every action available from the keyboard. Keep a visible focus indicator, and keep the focus order the same as the visual order.
- Give a switch input `role="switch"`.
- Give a resizer `role="slider"`, `tabindex="0"`, `aria-orientation`, `aria-valuenow`, `aria-valuemin`, and `aria-valuemax`. Resize it with the arrow keys. Do not use `role="separator"`, because the Svelte checker rejects a focusable separator.
- Announce notifications through one polite live region. Use `role="alert"` for an error that blocks the current task.

## Writing

- Use sentence case for all interface text, including buttons, headings, tabs, and menu items. The page names Accounts, Games, Logs Explorer, and Settings are proper names.
- Start a button label or menu item with a verb. Name the object when the context does not make it clear.
- Use the same verb for an action in every place: the button, the progress label, and the result. For example, use "Remove account", "Removing…", and "Account removed".
- Write an error as what happened, then what the user can do. Do not apologize or blame the user.
- Use the ellipsis character "…" in progress labels, not three periods.
- Use digits for numbers. Do not use exclamation marks.
- Keep labels to a few words and descriptions to one sentence.

Use these terms consistently:

- **Remove**: take an account, browser, or other item out of the application.
- **Delete**: permanently erase data, such as a tag or a file.
- **Close**: end Roblox processes or close a window. Do not use "kill".
- **Clear**: empty a field, search, filter, or selection.
- **Lock** and **Unlock**: change the vault state.
- **Launch**: start Roblox with one or more accounts.

## Motion

Motion must feel immediate. If an animation feels like something the user waits for, it is too slow.

Use these durations and easings:

- Hover color: `--duration-instant` (70ms) to `--duration-fast` (90ms), `--ease-standard`
- Press feedback: `--duration-instant` (70ms), `--ease-standard`
- Tooltip: `--duration-fast` (90ms), `--ease-sharp`
- Menu or popover enter: `--duration-ui` (110ms), `--ease-sharp`
- Menu or popover exit: `--duration-exit` (80ms), `--ease-standard`
- Dialog enter: `--duration-dialog` (130ms), `--ease-sharp`
- Dialog exit: `--duration-exit` (80ms), `--ease-standard`
- Small panel enter: `--duration-enter` (140ms) maximum, `--ease-sharp`
- Notification reflow: `--duration-enter` (140ms), `--ease-standard`

- No normal transition exceeds `160ms`.
- Animate movement only with `transform` and `opacity`, over small distances (`--motion-distance`, `--motion-scale`, `--press-scale`). Do not use bounce, spring overshoot, slow fades, or decorative loops.
- Notifications may slide to their new positions with `toastMove` from `presence.ts` when another notification enters or leaves. This slide is the only movement larger than `--motion-distance`.
- Direct hover and focus feedback may also transition `color`, `background-color`, and `border-color`.
- Do not animate layout properties, `filter`, `backdrop-filter`, blur, or `box-shadow`. Never use `transition: all`. List each transitioned property.
- The only continuous animation is a functional busy spinner. Do not run an idle `requestAnimationFrame` loop. Apply `will-change` only to an element that is about to animate.
- Keep menus, dialogs, tooltips, notifications, and operation errors mounted during their exit transition. Reopening a closing surface reverses or cancels the exit. It does not queue another animation. Restore focus only after a dialog finishes closing.
- When the selected account, settings category, or filtered settings result changes, use a restrained entrance for the new content. Do not animate the outgoing content. Do not stagger account rows or tag rows.
- Validation feedback may replay one horizontal nudge of at most 2px for 140ms. Cancel the previous nudge before replaying it. Do not hide or delay the error text.

## Reduced motion

The Animations setting in Settings > User Interface chooses System (the default), Reduced, or Full. System follows the operating system's `prefers-reduced-motion` preference.

`lib/shared/motion.ts` resolves the setting and sets the `data-reduced-motion` attribute on the root element. `base.css` defines the reduced-motion guard for that attribute, and `presence.ts` reads it through `reducedMotion()`. Define the guard only once, and do not query `prefers-reduced-motion` anywhere else. Under reduced motion:

- Exits complete immediately.
- The spinner stops but stays visible as a static busy indicator.
- Selected, expanded, loading, success, warning, and error states stay understandable without animation.
