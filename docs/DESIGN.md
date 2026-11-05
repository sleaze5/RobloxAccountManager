# Design

This file is the authoritative specification for visual design, component styling, interaction states, accessibility behavior, UI text, and motion. When another document conflicts with it on these topics, follow this file.

The interface is quiet, precise, technical, dense, and fast. It is a flat, near-black, almost grayscale UI with thin borders, small brightness steps, consistent geometry, and very short transitions. The content sits one small step darker than the chrome around it.

When this file does not cover a case, copy the closest existing pattern in the application. If no pattern fits, follow the description above and name the new pattern in the change summary.

## Sources of truth

The frontend is styled with Tailwind CSS 4. Markup carries utility classes. Stylesheets hold only the theme, global element styles, and shared building blocks.

- `frontend/src/styles/theme.css` defines the colors, fonts, text sizes, shadows, easings, and animations as Tailwind theme tokens. Spacing, sizes, radius, opacity, z-index, and durations use Tailwind's stock scales. See [Allowed values](#allowed-values).
- `frontend/src/styles/base.css` holds global element styles, the `motion-safe` and `motion-reduce` variants, and the reduced-motion guard.
- `frontend/src/styles/components.css` defines the shared building blocks as `@utility` classes. See [Building blocks](#building-blocks).
- `frontend/src/styles/app.css` is the entry stylesheet. It imports Tailwind, the theme, the base styles, and the building blocks.
- `frontend/scripts/check-styles.ts` enforces this file's styling rules. It runs with the lint task. Do not work around it.
- When TypeScript needs a token value, read it from the computed style. Do not repeat token values in TypeScript.
- `frontend/src/lib/shared/presence.ts` defines the enter and exit transitions for menus, dialogs, tooltips, and notifications that mount and unmount with `{#if}`. Reuse them.
- [`assets/icon.svg`](../assets/icon.svg) is the editable source of the application icon.

## Writing styles

- Style an element with utility classes in its `class` attribute. Write each class name in full, never built from parts, so that Tailwind can find it.
- Toggle classes with Svelte's array and object form, such as `class={["list-row", selected && "selected"]}`.
- Do not write `<style>` blocks in components. Do not write arbitrary values (`p-[13px]`), arbitrary properties (`[mask:none]`), or arbitrary variants (`[&>svg]:size-4`). Arbitrary values are allowed only inside `data-[…]` and `aria-[…]` variants.
- Pass only runtime values through inline styles, such as a sidebar width that the user drags. Set them with `style:` or as a CSS variable, and read a variable with Tailwind's parenthesized form, such as `w-(--sidebar-width)`. Do not read theme variables this way, and do not write literal sizes, colors, or durations in `style`.
- Put a repeated group of utilities that defines a role into a Svelte component when it has structure, or into an `@utility` in `components.css` when it styles a single element. Name a building block by its role, such as `list-row`, never by the feature that first used it.
- Write `@utility` blocks with theme variables and `--spacing(n)`. Do not write raw colors, pixel values above 2px, or durations in stylesheets other than `theme.css` and `base.css`.
- Add a theme token only for a value that defines the design and that the stock scales cannot express. Name tokens by role.

### Allowed values

`check-styles.ts` rejects any other value.

- Spacing and sizes (padding, margin, gap, width, height, size, inset, translate, leading): Tailwind's classic steps `0`, `px`, `0.5`, `1`, `1.5`, `2`, `2.5`, `3`, `3.5`, `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `14`, `16`, `20`, `24`, `28`, `32`, `36`, `40`, `44`, `48`, `52`, `56`, `60`, `64`, `72`, `80`, `96`, the named sizes (`full`, `auto`, `min`, `max`, `fit`, `screen`, and the container sizes such as `max-w-sm`), and fractions of 2, 3, 4, 5, 6, or 12.
- Opacity, and opacity modifiers on colors such as `bg-danger/10`: steps of 5.
- Text sizes: `text-2xs`, `text-xs`, `text-sm`, `text-base`, and `text-lg`, without a line-height modifier.
- Radius: `rounded-sm` (4px), `rounded-md` (6px), `rounded-lg` (8px), `rounded-xl` (12px), and `rounded-full`.
- Border, outline, and ring widths: `0`, `1`, `2`, `4`, and `8`, or the bare `border` (1px).
- z-index: `z-0`, `z-10`, `z-20`, `z-30`, `z-40`, `z-50`, and `z-auto`.
- Durations and delays: `75`, `100`, and `150`.
- Scale: `0`, `50`, `75`, `90`, `95`, `100`, `105`, and `110`. Rotation: `0`, `1`, `2`, `3`, `6`, `12`, `45`, `90`, and `180`.
- Not allowed: bare `transition`, `transition-all`, `animate-ping`, `animate-pulse`, and `animate-bounce`.

## Building blocks

Build from these parts before you write new styles. If a part lacks a variant you need, extend the part. Do not create a parallel style for the same role. Building blocks keep their descendant rules inside `:where()`, so a utility on an element overrides them.

Buttons (`@utility` classes):

- `btn-primary`: the one dominant action in a dialog, drawer, or empty state. The Launch button uses this style.
- `btn`: a bordered secondary button with a text label.
- `btn-icon`: a bordered icon-only button.
- `btn-ghost`: a borderless icon-only button. `btn-ghost-sm` is its 24px form.
- `btn-close`: the 24px close button of a dialog, notice, or error. It inherits the text color.
- `btn-danger` and `btn-warning`: confirmation actions with a danger or warning tone. See [Destructive actions](#destructive-actions).
- Set `data-open` on a button while the menu it opens is open.

Inputs:

- `field`: a text field. `search-field` wraps a search input with a leading icon and a clear button.
- `lib/shared/Select.svelte`: a dropdown. Use it instead of `<select>`. Pass its width with `className`.
- `lib/shared/NumberStepper.svelte`: a numeric input.
- `checkbox`: a checkbox. `check-option` is a checkbox with a label and an optional description.
- `switch`: a switch. Its input has `role="switch"`.
- `radio` inside `radio-option`: a radio button with its label.
- `segmented`: two to four mutually exclusive options that stay visible, as buttons with `aria-pressed` or as labels around radio inputs.
- `checklist`: requirements that turn valid or invalid as the user types, with `data-touched` and `data-valid`.

Surfaces and feedback:

- `menu`, `menu-item`, `menu-item-danger`, `menu-item-warning`, `menu-heading`, and `menu-separator`: menus and context menus. `submenu` opens beside a menu item on hover or focus.
- `dialog-backdrop`, `dialog-card`, `dialog-heading`, `dialog-field`, and `dialog-actions`: dialogs. See [Dialogs](#dialogs).
- `notice-warning`, `notice-error`, and `notice-note`: inline notices.
- `lib/shared/OperationError.svelte`: a failed operation that has no specific place on the page.
- `lib/notifications/notification-center.svelte.ts`: notifications.
- `lib/shared/Tooltip.svelte`: tooltips, through the `data-tooltip` attribute.
- `transfer` and `progress-bar`: download and install progress.
- `spinner`: the busy spinner on an icon.

Content:

- `empty-state`: empty, loading, and unavailable states for a page or pane.
- `tag-pill` and `tag-pill-favorite`: tags.
- `list-row`: a row in a selectable list, with `data-selected` and `data-open`. Show a drop position while dragging with `shadow-drop-before` or `shadow-drop-after`.
- `facts`: a list of labels and values, written as a `<dl>`. Set `data-tone` on a row for a success, warning, or muted value.
- `detail-grid`: compact label and value pairs in columns.
- `timeline`: a two-column timeline of markers and cards.
- `server-table`, `server-columns`, `server-row`, `server-number`, and `server-tag`: the game server tables.
- `text-block`: multi-line technical text.
- `lib/shared/Timestamp.svelte`, `lib/shared/CompactNumber.svelte`, and `lib/shared/SensitiveValue.svelte`: formatted dates, compact numbers, and hidden secrets.

Layout:

- `tab`: a page tab, with `aria-current="page"` on the selected tab.
- The workbench: the sidebar and detail pane layout of Accounts, Games, and Logs Explorer, with `lib/layout/SidebarResizer.svelte` as the resizer and `lib/layout/StatusBar.svelte` as the status strip.
- `settings-page`, `settings-header`, `settings-layout`, `settings-sidebar`, `settings-nav` with `nav-item`, and `settings-content`: Settings, account settings, chats, and other sub-pages with a navigation sidebar.
- `settings-section`, `settings-rows`, `settings-row`, and `settings-row-copy`: groups of settings.
- `scrollbar-stable`: reserve the scrollbar gutter in a scrolling list.

Scripts find elements through data attributes, never through class names, because utility classes change with the design.

## Embedded browser UI

The user must never see UI that looks or behaves like an exposed embedded browser. Everything the frontend shows must look intentionally designed as part of the application.

- After the frontend is ready, keep every user-facing interaction inside the WebView UI. Native operating-system UI is acceptable only during startup, before the frontend can show its own UI, and for the operating system's file dialogs when the user chooses where to save or which file to open.
- Do not use `alert()`, `confirm()`, or `prompt()`. Use the application's dialogs and notifications.
- Do not use the `title` attribute for tooltips. Use `data-tooltip` (and optionally `data-tooltip-side`, and `data-tooltip-strong` to bold one part of the text), which `lib/shared/Tooltip.svelte` renders.
- Add `novalidate` to every `<form>`. Show validation errors in the application UI, not in browser validation popups.
- The default context menu is disabled for the whole application. Show a context menu only where a custom one is implemented, such as the account list.
- Standard HTML controls are allowed only when they are styled to match the design. Replace a control that cannot be fully restyled.
- If no suitable building block exists, create one that follows this file.

## Layout and surfaces

- Design for a 900px by 500px window at 100% display scaling. This is also the minimum window size. Check every layout change at this size with the launch panel open at its default height.
- At the minimum size, the detail pane next to a default-width sidebar is about 680px wide. Make tables and fixed-column grids fit that width. Hide or truncate lower-priority columns before you add horizontal scrolling.
- Use `bg-surface` for persistent chrome (the title bar, page tabs, sidebars, tool rails, and status strip) and `bg-canvas` for the main content plane. The content is one small step darker than the chrome. Keep that step small and separate regions mainly with borders.
- Use `bg-overlay` for dialogs, menus, popovers, and other floating surfaces, which sit one step above the content. Use `bg-raised` for compact opaque controls. Inputs have no fill of their own.
- Separate regions with a background step plus a 1px border, not with large shadows. Use `border-line` by default, `border-line-subtle` between rows and shell regions, and `border-line-strong` for selected controls and floating surfaces that need extra separation. Avoid bright outlines.
- Keep the shell flat. The title bar and the page tabs are `h-9`, the status strip is `h-6`, and the default sidebar is `w-56`.
- Show each value once per screen. For example, do not show the same count in a sidebar header and in the status strip.

Accounts, Games, and Logs Explorer share one page anatomy:

1. The page tabs sit below the title bar.
2. A resizable, collapsible sidebar on `bg-surface` holds a header row (title, count, and icon actions), a search row, and a list.
3. A detail pane on `bg-canvas` shows the selected item, or an `empty-state` when nothing is selected.
4. A status strip at the bottom shows context for the current selection.

Settings replaces the page tabs with a header that has a back button.

Stack floating layers in this order, from lowest to highest:

1. Page content: `z-0` to `z-10` for sticky headers and overlapping parts of one component
2. Operation errors: `z-20`
3. Notifications: `z-30`
4. Dialogs: `z-40`
5. The vault lock screen: `z-50`

Menus, context menus, dropdowns, and tooltips open in the browser's top layer, above every layer in this list, so that a scroll area or the launch panel never cuts them off. Position a menu with `placeMenuAtAnchor`, `placeMenuAtPoint`, or the `floatMenu` action from `presence.ts`, which lift it automatically. Use `liftToTopLayer` for a floating surface that positions itself. Do not use `fixed` alone for a menu inside a scroll area, because `container-type` on an ancestor clips it.

## Typography

- Use Geist (`font-sans`) for interface text and Geist Mono (`font-mono`) for technical text such as paths, values, IDs, and measurements.
- Bundle both fonts as local, normal-style WOFF2 variable fonts from Vercel's pinned release, with weights 100 through 900. Preload Geist in `frontend/index.html`. Do not fetch fonts at runtime. Keep the pinned upstream sources in `THIRD_PARTY_NOTICES.md` and the font license in `THIRD_PARTY_LICENSES.txt`.
- Use the text scale: `text-2xs` (11px), `text-xs` (12px), `text-sm` (13px), `text-base` (15px), and `text-lg` (16px).
- Use `text-sm` for body text and list-row titles, and `text-xs` for controls, compact labels, and helper text. Use `text-2xs` for status-strip text, tooltips, badges, and compact technical text. Use `text-2xs` or `text-xs` monospace for technical text.
- Use `text-base font-semibold` for page titles, pane titles, and dialog titles. Use `text-sm font-semibold text-fg` for subheadings: sub-page titles, card titles, chat thread titles, and empty-state titles. Use `text-xs font-semibold text-fg-muted` for section headings inside a page or pane. Use `text-lg font-semibold` only for the name of the selected account or game in its detail header. Do not use other heading sizes.
- Use `font-normal` for body text. Use `font-medium` for buttons, menu items, control labels, selected navigation, and compact emphasis. Use `font-semibold` for headings, names, and strong labels. Use `font-bold` only for numeric count badges.
- Create hierarchy with weight, color, spacing, and grouping, not oversized text.

Text colors:

- Use `text-fg` for body text, names, and important labels.
- Use `text-fg-muted` for timestamps, helper text, metadata, and secondary labels.
- Use `text-fg-faint` for placeholders, inactive navigation, file paths, and tertiary information. Use it only on `bg-deep`, `bg-surface`, `bg-canvas`, and `bg-overlay`. On raised, hover, and selected fills, use `text-fg-muted` or a stronger color.
- Use `text-fg-bright` sparingly for primary buttons and the strongest emphasis.
- Do not use accent colors or opacity modifiers for text.

## Icons

- Use Phosphor icons when an appropriate icon exists. Import each icon as a single-weight raw SVG from `@phosphor-icons/core/<weight>/<name>.svg?raw`, for example `import gearIcon from "@phosphor-icons/core/regular/gear.svg?raw"`, and render it with `lib/shared/Icon.svelte`, for example `<Icon svg={gearIcon} size={16} aria-hidden="true" />`. Do not use barrel imports or an icon registry.
- Use the default `regular` weight, `light` for title-bar controls, `bold` for compact checkbox marks, and `fill` for filled states such as favorites. Pick the weight by importing its file (`fill/star-fill.svg?raw`), and import each weight you switch between. Do not override the SVG `fill` or stroke width.
- Use 16px icons by default in buttons, menu items, list rows, and fields.
- Use 14px icons in 24px controls, in tags, and next to 12px text. Use 12px icons next to 11px or smaller text. Use 11px icons for checkbox marks.
- Use 18px icons for title-bar controls and 28px icons for empty states.
- Use one icon for one meaning. Do not reuse a page's icon for an unrelated action, and do not use a trash icon for an action that only clears input.
- Add `aria-hidden="true"` to decorative icons. Keep accessible names on icon-only controls and meaningful standalone icons.

## Spacing and geometry

- Use the 4px spacing scale. Use `1` (4px) for tight internal spacing, `2` (8px) for compact controls, `3` (12px) for row and group gaps, and `4` (16px) for larger internal padding. Keep page and surface padding between `3` and `5`, and use `4` to `6` between major content groups.
- Prefer `gap` over margins for rows and columns.
- Use `size-6` (24px) controls for compact icon actions, `h-7` (28px) controls for ordinary buttons and menu rows, and `h-8` (32px) controls for text fields and dominant actions.
- Make list rows `h-12`, with `size-9` thumbnails. A row that also shows a game or place in its presence is `h-16`, with three lines: name, username, and presence.
- Keep nested curves concentric: child radius = outer radius - inset. A menu is `rounded-xl` with `p-1.5`, so its items are `rounded-md`.

Use the radius scale:

- `rounded-sm` (4px): chips, tags, badges, checkboxes, and segmented-control items
- `rounded-md` (6px): buttons, inputs, menu items, inline notices, avatars, and tooltips
- `rounded-lg` (8px): cards, panels, and text blocks
- `rounded-xl` (12px): menus, popovers, dialogs, notifications, and large surfaces

Use `rounded-full` for status dots, radio buttons, switch thumbs, and the switch track. Do not use other radius values, and do not turn other controls into pills.

## Controls and states

Inputs:

- Style inputs as outlines: a transparent background that takes the shade of the surface below, a 1px `border-line` border, `rounded-md`, and `text-fg-faint` placeholders. On focus, use `border-line-focus`. The `field` building block does this.
- Give every text field a visible label. With a visible label, the field border may stay subtle.

Buttons:

- Style the primary button with a `bg-raised` fill, a `border-line-strong` border, and `text-fg-bright` text. On hover, use `bg-raised-hover` and `border-line-focus`. Use at most one primary button in a dialog, drawer, or empty state.
- Keep secondary buttons quiet: transparent background, `border-line` border, and `text-fg-muted` text. On hover, use `bg-hover` and `text-fg`.
- Reserve `solid` and `on-solid` for checked checkboxes, count badges, and text selection. Do not use them for buttons.

Hover and selection:

- Keep hover and active feedback low contrast (`bg-hover`, `bg-active`). Do not use large brightness changes, and do not replace hover states with shadows.
- Mark the selected row in a list with two cues: a 2px `accent` bar on the left edge that grows in, and a `bg-active` fill. The bar carries the selection. Keep the fill quieter than `bg-hover`, because a brighter fill makes the list heavy.
- When the pointer is over a selected row, show the `bg-hover` fill and keep the bar.
- Draw the selection bar the same way in every list, with `list-row`. The bar must not shift the row content.
- Mark the selected page tab with a 2px `accent` bar on its bottom edge, drawn by the `tab` building block.
- Mark the selected item in a sidebar navigation list with a `bg-active` fill, a `border-line` border, and `font-medium`.

Disabled and other states:

- Every interactive element exposes the hover, focus, disabled, loading, and error states that apply to it.
- Show a disabled control with `opacity-45` and `cursor-not-allowed`. Do not also change its colors. Base styles do this for disabled buttons. When the reason is not obvious, explain it in visible text near the control.

Color:

- Keep the normal interface grayscale. Use `accent` and `accent-strong` only for indicators and badges, such as the selection bar, the tab bar, and filter counts.
- Use the success, warning, danger, and presence colors only to communicate meaning. Avoid large decorative status fills. Tint a surface or border with an opacity modifier, such as `border-danger/40` and `bg-danger/10`.
- Do not color an action by what it does, such as green for Join. Use semantic color on an action only when the action is destructive or needs a warning.
- Style inline code with the monospace font and the neutral `bg-fill` background. Do not color inline code like a link.

## Patterns

### Lists

- Give list rows the `list-row` building block, `h-12` (`h-16` when its presence names a game or place), and a 1px `border-line-subtle` separator. Set `aria-selected="true"` or `data-selected` on the selected row and `data-open` while its menu is open. `list-row` draws the hover and selection treatment from [Controls and states](#controls-and-states).
- When a list can be reordered, support dragging and `Ctrl+Shift+ArrowUp` and `Ctrl+Shift+ArrowDown`. Announce the new position through a polite live region.
- Distinguish an empty list ("No accounts yet") from a list with no matches ("No matching accounts"). Tell the user how to change each state.

### Menus

- Open a menu from its trigger on click. Open a context menu on right-click, `Shift+F10`, and the `ContextMenu` key.
- Move between menu items with the arrow keys, `Home`, and `End`. `Escape` closes the menu and returns focus to the trigger.
- Do not make right-click the only way to reach an important action. Also offer the action from a visible control, such as a More actions button.
- Build menus from `menu` and `menu-item`: 28px items with `text-xs` medium text, a 16px icon on the left, and a check on the right for a toggled item.
- Put destructive items last, after a `menu-separator`, with `text-danger`.

### Dialogs

1. Render the dialog as a `dialog-card` inside a `dialog-backdrop`. Set its width with a `max-w-*` container size. Put the title and description in a `dialog-heading`, fields in `dialog-field`, and actions in `dialog-actions`. Use `modalFocus` and the dialog transitions from `presence.ts`. `modalFocus` moves focus to the first text field, or to the first control when the dialog has no text field.
2. Use `role="dialog"`. Use `role="alertdialog"` for a confirmation.
3. Put the title and an optional one-sentence description at the top.
4. Put a `btn-close` in the top-right corner. Omit the close button only when the user must complete the dialog, such as the vault lock screen.
5. Show errors inside the dialog with `notice-error`, above the actions.
6. Put the actions at the bottom right: Cancel first, then the main action.
7. Name the main action with a verb and an object, such as "Remove account". Do not use "OK" or "Yes".
8. While an action runs, disable the dialog's controls and show the progress form of the label, such as "Removing…".
9. Make `Escape` and the close button behave like Cancel. While an action runs, they do nothing.

### Destructive actions

- Confirm every destructive action in a dialog that names the affected item and the consequence.
- Use `btn-danger` only for actions that delete data or end something that cannot resume. Use the primary style for a reversible action, such as locking the vault.
- State when an action cannot be undone.
- When an action can delete data that has no backup, require the user to type a confirmation word, such as `RESET`.

### Forms and validation

- Put a label above each field in `text-xs text-fg-muted`. Add "(optional)" to the label of an optional field.
- Validate a field after it loses focus or when the user submits the form. Do not show errors for a field the user has not changed.
- Show a field error below the field in `text-danger-muted`. Set `aria-invalid="true"` on the field. `field` then draws a `border-danger/40` border.
- Show the items of a requirement checklist in `text-fg-muted` until the user types. After that, mark each requirement as met or unmet with an icon and color.

### Loading, empty, and error states

- Show loading with a spinner and short text, such as "Loading accounts…". Set `aria-busy` on the region that is loading.
- When content refreshes, keep the existing content visible and replace the refresh icon with the spinner.
- Build an `empty-state` from a 28px icon, a title that says what is missing, one sentence that says how to fill it, and an action button when one exists.
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

- Follow [Capitalization](#capitalization).
- Start a button label or menu item with a verb. Name the object when the context does not make it clear.
- Use the same verb for an action in every place: the button, the progress label, and the result. For example, use "Remove account", "Removing…", and "Account removed".
- Write an error as what happened, then what the user can do. Do not apologize or blame the user.
- Use the ellipsis character "…" in progress labels, not three periods.
- Use digits for numbers. Do not use exclamation marks.
- Keep labels to a few words and descriptions to one sentence.

### Capitalization

Use sentence case by default. Capitalize only the first word and any word that is always capitalized.

- Use sentence case for buttons, menu items, headings, labels, descriptions, tooltips, notifications, placeholders, segmented options, and sidebar navigation items, such as "Remove account", "Account info", and "Search favorites".
- Use title case for page tab names: Accounts, Social, Games, and Logs Explorer. Settings is a page name too.
- Keep proper nouns in their own capitalization wherever they appear: product, service, and brand names, such as Roblox, Roblox Account Manager, Roblox Studio, RoValra, Chrome for Testing, Sober, Windows, and Linux. A proper noun inside a sentence-case label keeps its capitals, as in "Close Roblox processes".
- Write acronyms and initialisms in capitals, such as ID, PIN, URL, and API, as in "Place ID" and "User ID".
- Write user-provided text, such as names, nicknames, tags, and descriptions, exactly as the user or Roblox stores it.
- Write tokens, codes, and technical values in their exact case, such as `.ROBLOSECURITY`, `$YYYY`, and file names.

Use these terms consistently:

- **Remove**: take an account, browser, or other item out of the application.
- **Delete**: permanently erase data, such as a tag or a file.
- **Close**: end Roblox processes or close a window. Do not use "kill".
- **Clear**: empty a field, search, filter, or selection.
- **Lock** and **Unlock**: change the vault state.
- **Launch**: start Roblox with one or more accounts.

## Motion

Use motion wherever it makes the interface feel responsive: something appears or disappears, a state changes, an action completes, or a control is hovered. When you build or change a component, check its entrance, state changes, and action feedback, and add quiet motion where it fits. Motion must feel immediate. If the user waits for an animation, it is too slow.

Use these durations and easings:

- Hover color: `duration-100`, `ease-in-out`
- Press feedback: `duration-75`, `ease-in-out`
- Tooltip: `duration-100`, `ease-out`
- Menu or popover enter: `duration-100`, `ease-out`
- Menu, popover, or dialog exit: `duration-75`, `ease-in-out`
- Dialog enter: `duration-150`, `ease-out`
- Small panel enter: `duration-150` maximum, `ease-out`
- Notification reflow: `duration-150`, `ease-in-out`

`ease-out` is the sharp entrance curve from the theme. `presence.ts` reads the same durations from the `--duration-*` variables in `theme.css`.

### Patterns

These are examples, not a complete list.

- Surfaces: Animate a `popover` or `<dialog>` element with CSS: `starting:` for the entry state, `transition-discrete` for `display` and `overlay`, and `open:` for the open state. Use the `presence.ts` transitions for a surface that mounts and unmounts with `{#if}`.
- Exits: Keep menus, dialogs, tooltips, notifications, and operation errors mounted during their exit transition. Reopening a closing surface reverses or cancels the exit and does not queue another animation. Restore focus only after a dialog finishes closing.
- Changed content: When the selected account, settings category, or filtered settings result changes, use `animate-enter` on the new content. Do not animate the outgoing content.
- Added items: Use `animate-enter` on an item added to a list, such as a browser session or a tag.
- Indicators: Use `animate-pop` for an icon that appears in place or a status indicator that changes. Use `animate-bloom` for a check mark, a count badge, or a confirmation icon, such as Copied. Empty-state icons bloom in. Checkbox marks grow from `scale-50`.
- Selection: The selection bar of a `list-row` and the underline of a page `tab` grow in with `scale`.
- Hover: Mark a refresh icon with `data-turn` and a directional arrow or caret with `data-nudge="left"` or `data-nudge="right"`. Base styles turn the icon or nudge it 2px while the pointer is over its button.
- Disclosure: Rotate the caret 180 degrees with `transition-transform`, and use `animate-enter` on the revealed content.
- Press: An icon-only button may scale to `scale-95` while pressed.
- Progress: Grow a progress bar with `scale-x`, as `progress-bar` does.
- Notifications: Slide notifications to their new positions with `toastMove` from `presence.ts` when another notification enters or leaves.
- Validation: Replay one horizontal nudge of at most 2px for 150ms with `replayValidationNudge`. Cancel the previous nudge before replaying it. Do not hide or delay the error text.

### Limits

- No transition exceeds 150ms.
- Animate only `transform`, `translate`, `rotate`, `scale`, and `opacity`, so motion stays on the compositor. Hover and focus feedback may also transition `color`, `background-color`, and `border-color` with `transition-colors`.
- Do not animate layout properties, `filter`, `backdrop-filter`, blur, `box-shadow`, or color fills over large areas.
- Name the transitioned properties with `transition-colors`, `transition-opacity`, or `transition-transform`. Never use `transition` or `transition-all`.
- Move an element at most 2px. The `toastMove` slide is the only exception.
- Scale from 50% only small marks and badges, with `animate-bloom`. Scale larger surfaces from `scale-95` at most.
- Do not use bounce, spring overshoot, slow fades, decorative loops, or staggered rows.
- The only continuous animation is a functional busy spinner, `animate-spin`. Do not run an idle `requestAnimationFrame` loop. Apply `will-change` only to an element that is about to animate.

## Reduced motion

The Animations setting in Settings > User Interface chooses System (the default), Reduced, or Full. System follows the operating system's `prefers-reduced-motion` preference.

`lib/shared/motion.ts` resolves the setting and sets the `data-reduced-motion` attribute on the root element. `base.css` defines the reduced-motion guard for that attribute and redefines Tailwind's `motion-safe` and `motion-reduce` variants to follow it, and `presence.ts` reads it through `reducedMotion()`. Define the guard only once, and do not query `prefers-reduced-motion` anywhere else. Under reduced motion:

- Keep opacity and color transitions at their normal durations, so that state changes still fade instead of snapping.
- Remove all movement: transforms, scale, slides, hover nudges, and press feedback apply instantly or not at all.
- The spinner stops but stays visible as a static busy indicator: `motion-reduce:animate-none`.
- Selected, expanded, loading, success, warning, and error states stay understandable without animation.
