# Design

This file is the authoritative specification for visual design, component styling, interaction states, accessibility behavior, UI text, and motion. When another document conflicts with it on these topics, follow this file.

The interface is quiet, precise, technical, dense, and fast. It is a flat, near-black, almost grayscale UI with thin borders, small brightness steps, consistent geometry, and very short transitions. The content sits one small step darker than the chrome around it.

When this file does not cover a case, copy the closest existing pattern in the application. If no pattern fits, follow the description above and the [Design principles](#design-principles), and name the new pattern in the change summary.

## Design principles

Design the interface as one system. Take every value from a token, and judge every change against the full screen. The sections below set the concrete values. These principles tell you how to apply them and how to decide cases they do not cover. If a principle and a rule conflict, obey the rule.

1. **Geometric harmony.** Derive each radius, inset, border width, and proportion from adjacent elements. Keep nested corners concentric: inner radius = outer radius − padding − border width. Use one radius per component family, and only radii from the radius scale.
2. **Optical precision.** Align to perceived edges and visual weight, not to bounding boxes. Center asymmetric glyphs optically. Align adjacent text on one baseline. Balance icon size and stroke against adjacent text. Adjust padding until it looks even, with spacing-scale steps only.
3. **System-wide consistency.** Use one spacing scale, type scale, set of color roles, elevation order, radius scale, and motion set. A one-off value is a defect. Give a role the same appearance and behavior on all screens.
4. **Intentional microdetails.** Hairlines, separators, icon stroke, shadows, focus rings, and interaction states are part of the design. Small inconsistencies accumulate. Fix each one.
5. **Environmental harmony.** Make the interface feel native to a dense desktop tool: desktop density, window chrome, and pointer and keyboard conventions. Do not use web page patterns, such as hero sections, oversized cards, gradients, or glass surfaces.
6. **Refinement over decoration.** Give each element a function. Do not add gradients, glows, blur, decorative illustration, or decorative shadows. Make hierarchy with weight, color step, spacing, and grouping.
7. **Interaction craftsmanship.** Show clear cause and effect. Give feedback immediately and in proportion to the action. Keep spatial continuity: a surface enters from its trigger and exits toward it. Entrances decelerate, and exits are faster than entrances.
8. **Quality control.** Inspect each change at the real window size and in a 2× screenshot. Complete the [Design review checklist](#design-review-checklist) before you finish.

Nothing must look arbitrary or accidentally assembled. Each component must extend the system. Do not optimize one component at the cost of the full experience. Refine until the experience is coherent, precise, and complete.

## Sources of truth

The frontend is styled with Tailwind CSS 4. Markup carries utility classes. Stylesheets hold only the theme, global element styles, and shared building blocks.

- `frontend/src/styles/theme.css` defines the colors, fonts, text sizes, shadows, easings, and animations as Tailwind theme tokens. Spacing, sizes, radius, opacity, z-index, and durations use Tailwind's stock scales. See [Allowed values](#allowed-values).
- `frontend/src/styles/base.css` holds global element styles, the `motion-safe` and `motion-reduce` variants, and the reduced-motion guard.
- `frontend/src/styles/components.css` defines the shared building blocks as `@utility` classes. See [Building blocks](#building-blocks).
- `frontend/src/styles/app.css` is the entry stylesheet. It imports Tailwind, the theme, the base styles, and the building blocks.
- `frontend/scripts/check-styles.ts` enforces this file's styling rules. It runs with the lint task. Do not work around it.
- When TypeScript needs a token value, read it from the computed style. Do not repeat token values in TypeScript.
- `frontend/src/lib/shared/transitions.ts` defines the enter and exit transitions for menus, dialogs, tooltips, and notifications that mount and unmount with `{#if}`. Reuse them.
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
- Not allowed: bare `transition`, `transition-all`, `animate-ping`, `animate-pulse`, and `animate-bounce`. Loading placeholders pulse with `skeleton`, not `animate-pulse`.

## Building blocks

Build from these parts before you write new styles. If a part lacks a variant you need, extend the part. Do not create a parallel style for the same role. Building blocks keep their descendant rules inside `:where()`, so a utility on an element overrides them.

Buttons (`@utility` classes):

- `button-primary`: the one dominant action in a dialog, drawer, or empty state. The Launch button uses this style.
- `button`: a bordered secondary button with a text label.
- `button-icon`: a bordered icon-only button.
- `button-ghost`: a borderless icon-only button. `button-ghost-sm` is its 24px form.
- `button-close`: the 24px close button of a dialog, notice, or error. It inherits the text color.
- `button-text`: a borderless text button for a secondary action below a field, such as Show password or Recovery options.
- `button-danger` and `button-warning`: confirmation actions with a danger or warning tone. See [Destructive actions](#destructive-actions).
- Set `data-open` on a button while the menu it opens is open.

Inputs:

- `field`: a text field. `search-field` wraps a search input with a leading icon and a clear button.
- `lib/shared/Select.svelte`: a dropdown. Use it instead of `<select>`. Pass its width with `className`. Set `variant="status-strip"` for its compact, borderless form in a status strip.
- `lib/shared/NumberStepper.svelte`: a numeric input.
- `range`: a slider. Set `--range-fill` to the filled share of the track, such as `40%`, because the track cannot read the value of its input.
- `checkbox`: a checkbox. `check-option` is a checkbox with a label and an optional description. Render the check mark with `lib/shared/CheckboxMark.svelte`, and set `isIndeterminate` for a partial selection.
- `switch`: a switch. Its input has `role="switch"`.
- `radio` inside `radio-option`: a radio button with its label.
- `action-strip` with `lib/shared/RobloxActionStrip.svelte`: the bordered strip that holds every action that changes Roblox for the selected account, as icon-only buttons with a tooltip that names the action. Set `aria-pressed` on a toggled action and `data-tone` on a destructive one. Set `aria-disabled` instead of `disabled` on an action that Roblox refuses, and name the reason in its label, so that the keyboard reaches the reason.
- `segmented`: two to four mutually exclusive options that stay visible, as buttons with `aria-pressed` or as labels around radio inputs.
- `checklist`: requirements that turn valid or invalid as the user types, with `data-touched` and `data-valid`.

Surfaces and feedback:

- `menu`, `menu-item`, `menu-item-danger`, `menu-item-warning`, `menu-heading`, and `menu-separator`: menus and context menus. `submenu` opens beside a menu item on hover or focus.
- `dialog-backdrop`, `dialog-card`, `dialog-heading`, `dialog-field`, and `dialog-actions`: dialogs. See [Dialogs](#dialogs).
- `notice-warning` and `notice-note`: inline notices. Never show an error in a notice. See [Errors](#errors).
- `lib/notifications/notification-center.svelte.ts`: notifications.
- `lib/notifications/error-notifications.svelte.ts`: `notifyErrors`, which shows each error message of a component as an error notification.
- `lib/shared/Tooltip.svelte`: tooltips, through the `data-tooltip` attribute.
- `transfer` and `progress-bar`: download and install progress.
- `spinner`: the busy spinner on an icon. `lib/shared/RefreshIcon.svelte` shows the refresh icon, or the spinner while `isRefreshing` is true.
- `skeleton` and `skeleton-line`: placeholders for content that is loading. See [Loading, empty, and error states](#loading-empty-and-error-states). `skeleton` is a pulsing block that takes its size and radius from utilities. `skeleton-line` is as tall as a line of the surrounding text and paints a bar as tall as its font.
- `lib/shared/Skeleton.svelte`: one `skeleton-line` with a screen reader label. Pass its width with `className`.
- `lib/shared/SkeletonList.svelte`: placeholder rows for a list, in a `role="status"` region. `lib/shared/SkeletonMediaRow.svelte` is a row with an image and two lines, and `lib/shared/SettingsRowsSkeleton.svelte` is a set of `settings-row` placeholders.
- `lib/shared/SkeletonTile.svelte`: a placeholder item tile in a grid. Set `hasDetailLine={false}` when the real tiles show only a name.
- `lib/shared/LoadMore.svelte`: the end of a list that loads in pages. See [Lists](#lists).
- `lib/shared/MissingFactValue.svelte`: the value of a fact whose section has no value yet. It shows a skeleton while the section loads, "Unavailable" when it failed, and "N/A" or another empty label when the value does not apply.

Content:

- `empty-state`: empty, loading, and unavailable states for a page or pane.
- `lib/shared/MoreActionsMenu.svelte`: the More actions button at the top right of a profile or a details dialog, with its menu.
- `lib/shared/SavedAccountRelations.svelte`: the Saved accounts section of a user, a game, or an item, with badges that name how each other saved account relates to it.
- `lib/shared/AccountConfirmationActions.svelte`: the bottom row of a confirmation that changes Roblox or spends Robux, with the saved account that acts on the left and the actions on the right.
- `lib/shared/SettingChangeList.svelte`: the list of changed settings, each with its value before and after, that a confirmation shows before it saves.
- `tag-pill` and `tag-pill-favorite`: tags. `tag-pill-success` and `tag-pill-warning` are status pills that color only their icon.
- `list-row`: a row in a selectable list, with `data-selected` and `data-open`. Show a drop position while dragging with `shadow-drop-before` or `shadow-drop-after`.
- `facts`: a list of labels and values, written as a `<dl>`. Set `data-tone` on a row for a success, warning, or muted value.
- `detail-grid`: compact label and value pairs in columns.
- `timeline`: a two-column timeline of markers and cards.
- `server-table`, `server-columns`, `server-row`, and `server-tag`: the game server tables. Set `data-tone` on a `server-tag` for a success, warning, or danger status.
- `server-tiles`, `server-tile`, `server-stats`, and `server-health` with `lib/games/ServerTile.svelte`: the Roblox and RoValra server lists, one tile per server with its location, status tags, headline numbers, and expandable details. Set `data-density="compact"` on `server-stats` for the details, `data-tone` on a cell for a warning or danger value, and `data-font="sans"` on a value that is interface text.
- `text-block`: multi-line technical text.
- `lib/shared/Timestamp.svelte`, `lib/shared/CompactNumber.svelte`, and `lib/shared/SensitiveValue.svelte`: formatted dates, compact numbers, and hidden secrets.
- `lib/model-viewer/ModelFrame.svelte`: the one 3D viewer, with its toolbar for coordinates, reset view, and camera and lighting. Show every 3D model through it.
- `lib/model-viewer/AnimationControls.svelte`: the playback bar of a 3D avatar, with play and pause, the animation, the speed, and a seek bar.
- `frame-readout-top-left` and `frame-readout-bottom-left`: small technical readouts docked into a corner of a frame, such as camera coordinates.

Layout:

- `tab`: a page tab, with `aria-current="page"` on the selected tab.
- The workbench: the sidebar and detail pane layout of Accounts, Social, Groups, Games, Marketplace, and Logs Explorer, with `lib/layout/SidebarResizer.svelte` as the resizer and `lib/layout/StatusBar.svelte` as the status strip of Accounts.
- `lib/layout/AccountSwitcher.svelte`: the account switcher at the left of the status strip of Social, Groups, Games, Marketplace, and Logs Explorer.
- `settings-page`, `settings-header`, `settings-layout`, `settings-sidebar`, `settings-nav` with `nav-item`, and `settings-content`: Settings, account settings, chats, and other sub-pages with a navigation sidebar.
- `settings-section`, `settings-rows`, `settings-row`, and `settings-row-copy`: groups of settings. Put a setting that refines the setting before it, such as an update interval under its switch, in `settings-suboptions`.
- `lib/shared/SettingRow.svelte`: one setting in `settings-rows`, as a switch or a list of options, that shows the saved value until the change saves.
- `lib/shared/NoMatchingSettings.svelte`: the no-matches state of a settings search, with a Clear search button.
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
- Interface text is not selectable, because `base.css` sets `user-select: none` on `body`. Add `select-text` to content that people copy, such as names, IDs, values, descriptions, and messages. `facts`, `detail-grid`, `text-block`, notices, and `empty-state` messages already allow it. Ctrl+A selects only inside the focused text field or read-only textbox.
- If no suitable building block exists, create one that follows this file.

## Layout and surfaces

- Design for a 1080px by 600px window at 100% display scaling. This is the default and the minimum window size. Check every layout change at this size with the launch panel open at its default height.
- At the minimum size, the detail pane next to a default-width sidebar is about 830px wide, so a profile shows one column there and adds its right-hand column from `@4xl`. Make tables and fixed-column grids fit that width. Hide or truncate lower-priority columns before you add horizontal scrolling.
- Use `bg-surface` for persistent chrome (the title bar, page tabs, sidebars, tool rails, and status strip) and `bg-canvas` for the main content plane. The content is one small step darker than the chrome. Keep that step small and separate regions mainly with borders.
- Use `bg-overlay` for dialogs, menus, popovers, and other floating surfaces, which sit one step above the content. Use `bg-raised` for compact opaque controls. Inputs have no fill of their own.
- Separate regions with a background step plus a 1px border, not with large shadows. Use `border-line` by default, `border-line-subtle` between rows and shell regions, and `border-line-strong` for selected controls and floating surfaces that need extra separation. Avoid bright outlines.
- Keep the shell flat. The title bar and the page tabs are `h-9`, the status strip is `h-6`, and a page sidebar is 248px wide by default, so that its title fits a three-digit count beside four icon actions.
- Show each value once per screen. For example, do not show the same count in a sidebar header and in the status strip.
- Offer each destination and action once per screen. Do not add a button for a page that a tab already opens. A profile's More actions menu leaves out what the profile already shows as a button, a tab, or a link. A context menu on a list row keeps every action, because the row is where the pointer is.
- A pane that changes its layout with container variants, such as `@4xl:grid-cols-3`, sits inside an element with `@container`, usually its scroll region. Without one, the wide layout never applies. On a wide pane, a profile puts its main sections in the left two thirds and its secondary facts, such as the Vault of an account or the access and features of a game, in the right third.
- Only the content regions scroll, never the window. An absolutely positioned element, such as an `sr-only` live region, takes its position from the nearest positioned ancestor. Inside a scroll container, make the container `relative`, so that the element cannot overflow the window and let focus scroll the whole shell.

Accounts, Social, Groups, Games, Marketplace, and Logs Explorer share one page anatomy:

1. The page tabs sit below the title bar.
2. A resizable, collapsible sidebar on `bg-surface` holds a header row (title, count, and icon actions), a search row, and a list.
3. A detail pane on `bg-canvas` shows the selected item, or an `empty-state` when nothing is selected.
4. A status strip at the bottom shows context for the current selection.

Social uses the same status strip. Every page works with one selected account:

- Social, Groups, Games, Marketplace, and Logs Explorer show the selected account in the account switcher at the left of the status strip, with its 16px avatar, display name, and username. Switch the account there. Do not add another account picker to a page.
- Accounts selects in its list, so its status strip has no account switcher.
- An account stays selected whenever the vault has one. The first account in the list is selected when nothing else is. When the Accounts page closes, a multiple selection keeps only the selected account that is highest in the list.

Settings replaces the page tabs with a header that has a back button.

Stack floating layers in this order, from lowest to highest:

1. Page content: `z-0` to `z-10` for sticky headers and overlapping parts of one component
2. Dialogs: `z-40`
3. The vault lock screen: `z-50`
4. Notifications: `z-50`, after the lock screen in the document, so that a failure shows above every dialog

Menus, context menus, dropdowns, and tooltips open in the browser's top layer, above every layer in this list, so that a scroll area or the launch panel never cuts them off. Position a menu with `placeMenuAtAnchor`, `placeMenuAtPoint`, or the `floatMenu` action from `transitions.ts`, which lift it automatically. Use `liftToTopLayer` for a floating surface that positions itself. Do not use `fixed` alone for a menu inside a scroll area, because `container-type` on an ancestor clips it.

## Typography

- Use Geist (`font-sans`) for interface text and for prose that people write, such as descriptions, shouts, and messages, also inside a text field. Use Geist Mono (`font-mono`) for technical text: IDs and other technical numbers, such as versions, coordinates, and character counts, and paths, codes, and measurements. Headline numbers in `ProfileStats` stay in Geist.
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
- Make list rows `h-12`, with `size-9` thumbnails. A row that also shows a game or place in its online status is `h-16`, with three lines: name, username, and online status.
- Keep nested curves concentric: child radius = outer radius - inset. A menu is `rounded-xl` with `p-1.5`, so its items are `rounded-md`.

Use the radius scale:

- `rounded-sm` (4px): chips, tags, badges, checkboxes, and segmented-control items
- `rounded-md` (6px): buttons, inputs, menu items, inline notices, avatars, and tooltips
- `rounded-lg` (8px): cards, panels, and text blocks
- `rounded-xl` (12px): menus, popovers, dialogs, notifications, and large surfaces

Use `rounded-full` for status dots, radio buttons, switch thumbs, and the switch track. Do not use other radius values, and do not turn other controls into pills.

## Controls and states

Inputs:

- Style inputs as outlines: a transparent background that takes the shade of the surface below, a 1px `border-line` border, `rounded-md`, and `text-fg-faint` placeholders. On focus, use `border-line-focus`, which keeps a 3:1 contrast on every surface. The `field` building block does this.
- Draw the boundary of an unchecked checkbox, radio button, or switch track with `border-line-control`, which keeps a 3:1 contrast on every surface. The `checkbox`, `radio`, and `switch` building blocks do this.
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
- When a control's own action disables or removes it, such as a Reset button or the last Undo, move the focus to the next relevant control, such as the first control that it reset or the selected tab. Otherwise the keyboard focus drops to the page.
- Keep a Copy button enabled after it copies. Confirm the copy in the button with a check icon that blooms in and the label "Copied", as the Launch button does.

Color:

- Keep the normal interface grayscale. Use `accent` and `accent-strong` only for indicators and badges, such as the selection bar, the tab bar, and filter counts.
- Use the success, warning, danger, and online status colors only to communicate meaning. Avoid large decorative status fills. Tint a surface or border with an opacity modifier, such as `border-danger/40` and `bg-danger/10`.
- Do not color an action by what it does, such as green for Join. Use semantic color on an action only when the action is destructive or needs a warning.
- Style inline code with the monospace font and the neutral `bg-fill` background. Do not color inline code like a link.

## Patterns

### Lists

- Give list rows the `list-row` building block, `h-12` (`h-16` when its online status names a game or place), and a 1px `border-line-subtle` separator. Set `aria-selected="true"` or `data-selected` on the selected row and `data-open` while its menu is open. `list-row` draws the hover and selection treatment from [Controls and states](#controls-and-states).
- Make a list one tab stop. Give the selected row `tabindex="0"`, or the first row when no row is selected, and every other row `tabindex="-1"`. Mark the list container with `data-list` and its rows with `data-list-item`, and move between them with `ArrowUp`, `ArrowDown`, `Home`, and `End` through `moveListFocus`.
- When a list can be reordered, support dragging and `Ctrl+Shift+ArrowUp` and `Ctrl+Shift+ArrowDown`. Announce the new position through a polite live region.
- When an account needs the user's attention, its row adds a third line in the warning or danger tone, such as "Sign-in required", "Security challenge", or a moderation status, in place of its online status.
- Load a list that comes in pages as one growing list, never as pages to step through. End it with `LoadMore`, inside its scrolling container marked with `data-load-more-root`. `LoadMore` loads the next page when the end of the list comes within 160px of view, and always shows a "Load more" button with the noun, such as "Load more members", in case the automatic load does not run. While a page loads, it shows the spinner and the progress label, such as "Loading more members…". After a failed page, it stops loading on its own until the user presses the button, and shows "Next page unavailable" above the button. The list reports the cause as an error notification. Show the count of loaded items with "+" while more pages remain, and keep the items that are already loaded when a later page fails.
- Distinguish an empty list ("No accounts yet") from a list with no matches ("No matching accounts"). Tell the user how to change each state. Give the no-matches state a button that clears the filter and moves the focus to the search field or filter.
- In a grid of item tiles, make every tile in a row the same height. Keep the badges on a tile in one row, so that they never overlap, and let the lower-priority badge truncate.
- When the user switches a tab or category that replaces the content of a scrolling region, scroll the region back to the top.
- Settings search reads its words from `lib/settings/settings-search.ts`. Give each new setting an entry with its label, its description, and other words people use for the same thing, and add the entry to its category. The account settings pages keep their search words next to each section.

### Menus

- Open a menu from its trigger on click. Open a context menu on right-click, `Shift+F10`, and the `ContextMenu` key.
- Move between menu items with the arrow keys, `Home`, and `End`. `Escape` closes the menu and returns focus to the trigger.
- Do not make right-click the only way to reach an important action. Also offer the action from a visible control, such as a More actions button.
- Build menus from `menu` and `menu-item`: 28px items with `text-xs` medium text, a 16px icon on the left, and a check on the right for a toggled item.
- Put destructive items last, after a `menu-separator`, with `text-danger`.

### Dialogs

1. Render the dialog as a `dialog-card` inside a `dialog-backdrop`. Set its width with a `max-w-*` container size. Put the title and description in a `dialog-heading`, fields in `dialog-field`, and actions in `dialog-actions`. Use `modalFocus` and the dialog transitions from `transitions.ts`. `modalFocus` moves focus to the first text field, or to the first control when the dialog has no text field. When a region inside the dialog scrolls, such as an item grid, set `hasFlexibleBody` on `lib/shared/Dialog.svelte` and give the region `min-h-0`, so that the heading and the actions stay in view.
2. Use `role="dialog"`. Use `role="alertdialog"` for a confirmation.
3. Put the title and an optional one-sentence description at the top.
4. Put a `button-close` in the top-right corner. Omit the close button only when the user must complete the dialog, such as the vault lock screen.
5. Report a failed action as an error notification, and keep the dialog open so that the user can try again. Do not show the error inside the dialog.
6. Put the actions at the bottom right: Cancel first, then the main action.
7. Name the main action with a verb and an object, such as "Remove account". Do not use "OK" or "Yes".
8. While an action runs, disable the dialog's controls and show the progress form of the label, such as "Removing…".
9. Make `Escape` and the close button behave like Cancel. While an action runs, they do nothing.

### Destructive actions

- Confirm every destructive action in a dialog that names the affected item and the consequence.
- Use `button-danger` only for actions that delete data or end something that cannot resume. Use the primary style for a reversible action, such as locking the vault.
- State when an action cannot be undone.
- When an action can delete Roblox Account Manager data that has no backup, such as the vault, require the user to type a confirmation word, such as `RESET`. Actions on the data of other programs, such as cleaning or uninstalling Roblox, need only the confirmation dialog.

### Forms and validation

- Put a label above each field in `text-xs text-fg-muted`. Add "(optional)" to the label of an optional field.
- Validate a field after it loses focus or when the user submits the form. Do not show errors for a field the user has not changed.
- Show a field error below the field in `text-danger-muted` only for a check that the frontend runs on the input, such as a price range or a date format. Set `aria-invalid="true"` on the field. `field` then draws a `border-danger/40` border. Report a failure that the backend returns as an error notification.
- Show the items of a requirement checklist in `text-fg-muted` until the user types. After that, mark each requirement as met or unmet with an icon and color.

### Loading, empty, and error states

- Never let loading move the layout. Content that arrives must take the place of a placeholder of the same size, not push other content. For example, a 3D frame shows its playback bar disabled while the avatar loads, instead of adding the bar after the model appears.
- Show the first load of content as a skeleton shaped like that content: `skeleton-line` for text, `skeleton` blocks for images and controls, and `SkeletonList` rows as tall as the real rows. Show the parts that are already known, such as labels, headings, and setting descriptions, as real text, and use skeletons only for the values.
- Keep each row of a facts list, details pane, or settings list in place from the first frame. While its value loads, show a skeleton. When the value does not apply, show "N/A" in `text-fg-muted`. Use "None" when the account has nothing of that kind. When the value failed to load, show "Unavailable" in `text-fg-muted`. When the value applies but is not known, show "Unknown". When a log or build has no record of it, show "Not recorded". When Roblox hides it, show "Not shared by Roblox". `MissingFactValue` does this for section-based data.
- Give the loading region `role="status"` or `aria-busy="true"`, and give every skeleton a screen reader label, as `Skeleton` and `SkeletonList` do.
- Use the spinner for actions in progress: a pressed button, a refresh, a list that loads more rows below its existing rows, and a pane that waits for another region to load.
- When content refreshes, keep the existing content visible and replace the refresh icon with the spinner.
- Build an `empty-state` from a 28px icon, a title that says what is missing, one sentence that says how to fill it, and an action button when one exists.
- When content fails to load, never show its empty state, such as "No accounts yet". Show an `empty-state` with the regular `warning` icon, never the page's own icon, a title such as "Accounts unavailable", and a Try again button. Do not show the error message there. Report it as described in [Errors](#errors). When earlier content is still shown, keep it.
- When a smaller region fails, such as the details of an outfit, show a short `text-fg-muted` label, such as "Outfit details unavailable", and a Try again button when one helps.

### Errors

Never show error text in the page: no error notices, error lines below content, or error messages in empty states and dialogs. Each failure either shows an error notification or fails silently.

- Show an error notification when the user must know about the failure: a failed action, a failed load of a page, list, or dialog, a failed page of a list, and a failed search. Call `notifyErrors` with the notification title and a function that reads the error message, such as `notifyErrors("Badges not loaded", () => badges.error)`.
- Fail silently when the page already shows that the content is missing and the user loses nothing: a section whose facts show "Unavailable", secondary details in a dialog, a background refresh, and an image that does not load. The backend logs the failure.
- `notifyErrors` uses the message as the notification ID. When two components report the same failure, one notification shows. Clear an error before the action runs again, so that a repeated failure shows again.
- A rate-limit message names the wait only when the Roblox response sends one, such as "Roblox is limiting requests. Try again in 30 seconds." It never counts down.

### Tooltips

- Write tooltip text in sentence case without a final period. Keep it short.
- On an icon-only control, make the tooltip match or expand the accessible name.
- Do not put information only in a tooltip, because screen readers do not read tooltip text. Put required information in the accessible name, a description, or visible text.
- Close the tooltip on `Escape`.

### Notifications

- Show notifications only through the notification center.
- Notifications appear as snackbars at the bottom center, above the status strip. When more than 3 are open, the older ones collapse behind a "Show N more" button.
- Give each notification a short title and one sentence. Use a past-tense title for a result, such as "Accounts saved".
- Give an error notification an action that helps the user recover when one exists. An error notification without an action stays for 8 seconds, and pauses while the pointer or focus is on it.

## Accessibility

- Give all text a contrast ratio of at least 4.5:1 against its background.
- Give meaningful icons, state indicators, focus rings, and the boundaries of checkboxes, radio buttons, and switches a contrast ratio of at least 3:1 against their background.
- Do not rely on color alone. Pair semantic color with text, an icon, or another cue.
- Use `--color-axis-x`, `--color-axis-y`, and `--color-axis-z` only to mark the X, Y, and Z axes. Give each value shown in an axis color its axis name for screen readers.
- Make every action available from the keyboard. Keep a visible focus indicator, and keep the focus order the same as the visual order.
- Give a switch input `role="switch"`.
- Give a resizer `role="slider"`, `tabindex="0"`, `aria-orientation`, `aria-valuenow`, `aria-valuemin`, and `aria-valuemax`. Resize it with the arrow keys. Do not use `role="separator"`, because the Svelte checker rejects a focusable separator.
- Announce notifications, including errors, through one polite live region. Use `role="alert"` only for a field error that blocks the form.
- Do not skip heading levels. The page or profile title is the `h1`, visually hidden when the tabs already name the page. Each tab panel or pane starts at `h2`, and its groups use `h3`. When the selected tab names a panel, give the panel a visually hidden `h2` with the tab label.
- Give each page one banner and one `main` landmark, labeled with the page name, that holds the sidebar, the content, and the resizer between them. The window title bar is the only top-level `header`. An account sub-page renders inside the `main` of Accounts, so it uses labeled `section` regions instead of another `main` or an `aside`. Give a container with an `aria-label` a role, such as `role="group"` on a list.

## Writing

These rules add UI rules to [WRITING.md](./WRITING.md), which applies to all text.

- Follow [Capitalization](#capitalization).
- Use the terms and action verbs in [GLOSSARY.md](./GLOSSARY.md).
- Start a button label or menu item with a verb. Name the object when the context does not make it clear.
- Use the same verb for an action in every place: the button, the progress label, and the result. For example, use "Remove account", "Removing…", and "Account removed".
- Write an error as what happened, then what the user can do. Do not apologize or blame the user.
- Write errors in plain, concise language inspired by ASD-STE100 Simplified Technical English: short sentences, the active voice, one instruction in each sentence, and common words. Name the exact problem so that a technical user understands it, such as the Roblox limit, the missing file, or the rejected cookie. Do not show endpoints, HTTP status text, or internal error text.
- Use the ellipsis character "…" only at the end of a progress label, such as "Saving…" or "Loading accounts…", and never three periods. Do not use it in any other text: not in button labels or menu items, even when they open a dialog, and not in placeholders, examples, or truncation markers.
- Use digits for numbers. Do not use exclamation marks.
- Keep labels to a few words and descriptions to one sentence.
- Write short sentences in the active voice, with one idea in each. Shorten the wording, not the information: keep every detail the user needs, such as codes, statuses, flag names, and formats.
- Do not use contractions. Use US spelling, such as "canceled".
- Write an error as "Could not" with a verb and the thing that failed, then the recovery, such as "Could not load the badges. Try again." Use these recovery phrases: "Try again.", "Try again later.", "Refresh to try again.", and "Refresh to check." for a change that Roblox did not confirm. Use a specific instruction when one helps more, such as "Unlock the vault first."
- Title a failure notification with the noun and "not" with the past participle, such as "Browser not opened" or "Data not moved".
- Title a page or pane that failed to load with the noun and "unavailable", such as "Badges unavailable".
- Write a loading label as "Loading" with the noun and the ellipsis, such as "Loading badges…".
- Separate inline facts with " · ", and put ":" before a detail, such as "Paid access: R$ 100". Do not use " - " as a dash. Write "and", not "&".
- Give the same action the same label everywhere: "Close dialog" for the close button of a dialog, "More actions" for an overflow button, "Clear search" for the clear button of a search field, and "Show in Games" for opening a place in Games. A menu item and the dialog that it opens use the same words, such as "Sign in with browser".

### Capitalization

Use sentence case by default. Capitalize only the first word and any word that is always capitalized.

- Use sentence case for buttons, menu items, headings, labels, descriptions, tooltips, notifications, placeholders, segmented options, and sidebar navigation items, such as "Remove account", "Account details", and "Search favorites".
- Use title case for page tab names: Accounts, Social, Groups, Games, Marketplace, and Logs Explorer. Settings is a page name too.
- Keep proper nouns in their own capitalization wherever they appear: product, service, and brand names, such as Roblox, Roblox Account Manager, Roblox Studio, RoValra, Chrome for Testing, Sober, Windows, and Linux. A proper noun inside a sentence-case label keeps its capitals, as in "Close Roblox processes".
- Write acronyms and initialisms in capitals, such as ID, PIN, URL, and API, as in "Place ID" and "User ID".
- Write user-provided text, such as names, nicknames, tags, and descriptions, exactly as the user or Roblox stores it.
- Write tokens, codes, and technical values in their exact case, such as `.ROBLOSECURITY`, `$YYYY`, and file names.

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

`ease-out` is the sharp entrance curve from the theme. `transitions.ts` reads the same durations from the `--duration-*` variables in `theme.css`.

### Patterns

These are examples, not a complete list.

- Surfaces: Animate a `popover` or `<dialog>` element with CSS: `starting:` for the entry state, `transition-discrete` for `display` and `overlay`, and `open:` for the open state. Use the `transitions.ts` transitions for a surface that mounts and unmounts with `{#if}`.
- Exits: Keep menus, dialogs, tooltips, notifications, and operation errors mounted during their exit transition. Reopening a closing surface reverses or cancels the exit and does not queue another animation. Restore focus only after a dialog finishes closing.
- Changed content: When the selected account, settings category, or filtered settings result changes, use `animate-enter` on the new content. Do not animate the outgoing content.
- Added items: Use `animate-enter` on an item added to a list, such as a browser session or a tag.
- Indicators: Use `animate-pop` for an icon that appears in place or a status indicator that changes. Use `animate-bloom` for a check mark, a count badge, or a confirmation icon, such as Copied. Empty-state icons bloom in. Checkbox marks grow from `scale-50`.
- Selection: The selection bar of a `list-row` and the underline of a page `tab` grow in with `scale`.
- Hover: Mark a refresh icon with `data-turn` and a directional arrow or caret with `data-nudge="left"` or `data-nudge="right"`. Base styles turn the icon or nudge it 2px while the pointer is over its button.
- Disclosure: Rotate the caret 180 degrees with `transition-transform`, and use `animate-enter` on the revealed content.
- Press: An icon-only button may scale to `scale-95` while pressed.
- Progress: Grow a progress bar with `scale-x`, as `progress-bar` does.
- Notifications: Slide notifications to their new positions with `toastMove` from `transitions.ts` when another notification enters or leaves.
- Validation: Replay one horizontal nudge of at most 2px for 150ms with `replayValidationNudge`. Cancel the previous nudge before replaying it. Do not hide or delay the error text.

### Limits

- No transition exceeds 150ms.
- Animate only `transform`, `translate`, `rotate`, `scale`, and `opacity`, so motion stays on the compositor. Hover and focus feedback may also transition `color`, `background-color`, and `border-color` with `transition-colors`.
- Do not animate layout properties, `filter`, `backdrop-filter`, blur, `box-shadow`, or color fills over large areas.
- Name the transitioned properties with `transition-colors`, `transition-opacity`, or `transition-transform`. Never use `transition` or `transition-all`.
- Move an element at most 2px. The `toastMove` slide is the only exception.
- Scale from 50% only small marks and badges, with `animate-bloom`. Scale larger surfaces from `scale-95` at most.
- Do not use bounce, spring overshoot, slow fades, decorative loops, or staggered rows.
- The only continuous animations are the functional loading indicators: the busy spinner, `animate-spin`, and the skeleton pulse, `animate-skeleton`, which fades a skeleton's opacity in a 1200ms loop. Do not run an idle `requestAnimationFrame` loop. Apply `will-change` only to an element that is about to animate.

## Reduced motion

The Animations setting in Settings > User Interface chooses System (the default), Reduced, or Full. System follows the operating system's `prefers-reduced-motion` preference.

`lib/shared/motion.ts` resolves the setting and sets the `data-reduced-motion` attribute on the root element. `base.css` defines the reduced-motion guard for that attribute and redefines Tailwind's `motion-safe` and `motion-reduce` variants to follow it, and `transitions.ts` reads it through `reducedMotion()`. Define the guard only once, and do not query `prefers-reduced-motion` anywhere else. Under reduced motion:

- Keep opacity and color transitions at their normal durations, so that state changes still fade instead of snapping.
- Remove all movement: transforms, scale, slides, hover nudges, and press feedback apply instantly or not at all.
- The spinner and the skeleton pulse stop but stay visible as static loading indicators: `motion-reduce:animate-none`.
- Selected, expanded, loading, success, warning, and error states stay understandable without animation.

## Design review checklist

Review the full screen, then each component in each state. Use a 1080px by 600px window at 100% scaling, then a 2× screenshot.

- [ ] **Tokens:** All values come from tokens or stock scales.
- [ ] **Patterns:** The change reuses building blocks. A new pattern has a name and a reason.
- [ ] **Geometry:** Each radius matches its role, and nested corners are concentric. Borders are 1px, or 2px for indicators.
- [ ] **Proportions:** Control, row, and thumbnail sizes match across adjacent components.
- [ ] **Spacing:** Related items are closer than unrelated items. Sibling surfaces use the same padding.
- [ ] **Alignment:** Edges share axes across components, with no 1px or 2px near-misses. Adjacent text shares a baseline.
- [ ] **Layout:** The layout holds at the minimum window size, truncates in priority order, and does not shift on load.
- [ ] **Typography:** Size, weight, and color match the role. The hierarchy reads in a squint test.
- [ ] **Optical balance:** Icons look centered, and their weight matches adjacent text. Controls in a group have equal visual weight.
- [ ] **Color and elevation:** Surfaces follow the elevation order. Contrast meets [Accessibility](#accessibility). Semantic color shows meaning only.
- [ ] **States:** Each applicable state exists and matches across components: hover, pressed, focus-visible, selected, disabled, loading, empty, success, and error. Focus rings are not clipped.
- [ ] **Motion:** Feedback is immediate. Duration and easing match the role. Surfaces enter from their trigger. States stay clear under reduced motion.
- [ ] **Restraint:** Each element has a function. No decoration, redundant labels, or repeated values. At most one dominant action per surface.
- [ ] **Coherence:** The screen feels like a desktop application and matches its neighbors in anatomy, alignment axes, and spacing rhythm.
