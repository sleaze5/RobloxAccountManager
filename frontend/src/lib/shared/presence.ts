import type { EasingFunction, TransitionConfig } from "svelte/transition"
import { reducedMotion } from "./motion"

interface MotionOptions {
	duration: number
	easing: EasingFunction
	x?: number
	y?: number
	scale?: number
	origin?: string
}

const sharp = cubicBezier(0.16, 1, 0.3, 1),
	standard = cubicBezier(0.4, 0, 0.2, 1)

function motion({
	duration,
	easing,
	x = 0,
	y = 0,
	scale = 1,
	origin = "center",
}: MotionOptions): TransitionConfig {
	return {
		css: (t, u) =>
			`opacity: ${t}; transform: translate(${u * x}px, ${u * y}px) scale(${scale + (1 - scale) * t}); transform-origin: ${origin};`,
		duration: reducedMotion() ? 0 : duration,
		easing,
	}
}

export function menuIn(node: Element): TransitionConfig {
	const distance = tokenNumber(node, "--motion-distance", 2)
	return motion({
		duration: tokenDuration(node, "--duration-ui", 110),
		easing: sharp,
		origin:
			getComputedStyle(node).getPropertyValue("--menu-origin").trim() ||
			"top right",
		scale: tokenNumber(node, "--motion-scale", 0.99),
		y: distance,
	})
}

export function menuOut(node: Element): TransitionConfig {
	const distance = tokenNumber(node, "--motion-distance", 2)
	return motion({
		duration: tokenDuration(node, "--duration-exit", 80),
		easing: standard,
		origin:
			getComputedStyle(node).getPropertyValue("--menu-origin").trim() ||
			"top right",
		scale: tokenNumber(node, "--motion-scale", 0.99),
		y: distance / 2,
	})
}

export function modalBackdropIn(node: Element): TransitionConfig {
	return motion({
		duration: tokenDuration(node, "--duration-dialog", 130),
		easing: sharp,
	})
}

export function modalBackdropOut(node: Element): TransitionConfig {
	return motion({
		duration: tokenDuration(node, "--duration-exit", 80),
		easing: standard,
	})
}

export function modalCardIn(node: Element): TransitionConfig {
	return motion({
		duration: tokenDuration(node, "--duration-dialog", 130),
		easing: sharp,
		scale: tokenNumber(node, "--motion-scale", 0.99),
		y: tokenNumber(node, "--motion-distance", 2),
	})
}

export function modalCardOut(node: Element): TransitionConfig {
	const distance = tokenNumber(node, "--motion-distance", 2)
	return motion({
		duration: tokenDuration(node, "--duration-exit", 80),
		easing: standard,
		scale: tokenNumber(node, "--motion-scale", 0.99),
		y: distance / 2,
	})
}

export function tooltipIn(node: Element): TransitionConfig {
	return motion({
		duration: tokenDuration(node, "--duration-fast", 90),
		easing: sharp,
		scale: tokenNumber(node, "--motion-scale", 0.99),
		y: tokenNumber(node, "--motion-distance", 2),
	})
}

export function tooltipOut(node: Element): TransitionConfig {
	const distance = tokenNumber(node, "--motion-distance", 2)
	return motion({
		duration: tokenDuration(node, "--duration-instant", 70),
		easing: standard,
		scale: tokenNumber(node, "--motion-scale", 0.99),
		y: distance / 2,
	})
}

export function toastIn(node: Element): TransitionConfig {
	const distance = tokenNumber(node, "--motion-distance", 2)
	return motion({
		duration: tokenDuration(node, "--duration-enter", 140),
		easing: sharp,
		scale: tokenNumber(node, "--motion-scale", 0.99),
		y: -distance,
	})
}

export function toastOut(node: Element): TransitionConfig {
	const distance = tokenNumber(node, "--motion-distance", 2)
	return motion({
		duration: tokenDuration(node, "--duration-exit", 80),
		easing: standard,
		scale: tokenNumber(node, "--motion-scale", 0.99),
		y: -distance / 2,
	})
}

export { toastIn as errorIn, toastOut as errorOut }

export function replayValidationNudge(element: HTMLElement | undefined): void {
	if (!element || reducedMotion()) {
		return
	}
	element.getAnimations().forEach((animation) => {
		animation.cancel()
	})
	element.animate(
		[
			{ transform: "translateX(0)" },
			{ offset: 0.35, transform: "translateX(-2px)" },
			{ offset: 0.7, transform: "translateX(2px)" },
			{ transform: "translateX(0)" },
		],
		{
			duration: tokenDuration(element, "--duration-enter", 140),
			easing: "cubic-bezier(0.4, 0, 0.2, 1)",
		},
	)
}

const menuEdge = 8,
	menuGap = 4

type MenuAlign = "start" | "center" | "end"

export function liftToTopLayer(node: HTMLElement): void {
	if (typeof node.showPopover !== "function" || node.matches(":popover-open")) {
		return
	}
	node.popover = "manual"
	node.showPopover()
}

export function floatMenu(
	node: HTMLElement,
	options: { anchor: Element | undefined; align?: MenuAlign },
): { update: (next: typeof options) => void; destroy: () => void } {
	let current = options
	const place = () => {
		if (current.anchor?.isConnected) {
			placeMenuAtAnchor(node, current.anchor, current.align)
		}
	}
	place()
	window.addEventListener("resize", place)
	window.addEventListener("scroll", place, true)
	return {
		update: (next) => {
			current = next
			place()
		},
		destroy: () => {
			window.removeEventListener("resize", place)
			window.removeEventListener("scroll", place, true)
		},
	}
}

export function placeMenuAtPoint(
	node: HTMLElement,
	point: { x: number; y: number },
): { x: number; y: number } {
	liftToTopLayer(node)
	const x = Math.max(
			menuEdge,
			Math.min(point.x, window.innerWidth - node.offsetWidth - menuEdge),
		),
		y = Math.max(
			menuTop(node),
			Math.min(point.y, window.innerHeight - node.offsetHeight - menuEdge),
		)
	setMenuPosition(
		node,
		x,
		y,
		`${y < point.y ? "bottom" : "top"} ${x < point.x ? "right" : "left"}`,
	)
	return { x, y }
}

export function placeMenuAtAnchor(
	node: HTMLElement,
	anchor: Element,
	align: MenuAlign = "end",
): void {
	liftToTopLayer(node)
	const bounds = anchor.getBoundingClientRect(),
		opensUp =
			bounds.bottom + menuGap + node.offsetHeight > window.innerHeight - menuEdge,
		preferredX =
			align === "start"
				? bounds.left
				: align === "center"
					? bounds.left + bounds.width / 2 - node.offsetWidth / 2
					: bounds.right - node.offsetWidth,
		x = Math.max(
			menuEdge,
			Math.min(preferredX, window.innerWidth - node.offsetWidth - menuEdge),
		),
		y = Math.max(
			menuTop(node),
			opensUp
				? bounds.top - node.offsetHeight - menuGap
				: bounds.bottom + menuGap,
		)
	setMenuPosition(
		node,
		x,
		y,
		`${opensUp ? "bottom" : "top"} ${align === "start" ? "left" : align === "center" ? "center" : "right"}`,
	)
}

function menuTop(node: Element): number {
	return tokenNumber(node, "--titlebar-height", 38) + menuGap
}

function setMenuPosition(
	node: HTMLElement,
	x: number,
	y: number,
	origin: string,
): void {
	node.style.left = `${x}px`
	node.style.top = `${y}px`
	node.style.setProperty("--menu-origin", origin)
}

function tokenDuration(node: Element, token: string, fallback: number): number {
	return tokenNumber(node, token, fallback)
}

function tokenNumber(node: Element, token: string, fallback: number): number {
	const value = Number.parseFloat(getComputedStyle(node).getPropertyValue(token))
	return Number.isFinite(value) ? value : fallback
}

export function modalFocus(node: HTMLElement): { destroy: () => void } {
	const previous =
			document.activeElement instanceof HTMLElement
				? document.activeElement
				: null,
		focusableSelector =
			'button:not(:disabled), input:not(:disabled), textarea:not(:disabled), select:not(:disabled), [tabindex]:not([tabindex="-1"])',
		fieldSelector =
			'input:not(:disabled):not([type="checkbox"], [type="radio"], [type="hidden"]), textarea:not(:disabled)'

	queueMicrotask(() => {
		if (node.contains(document.activeElement)) return
		const target =
			node.querySelector<HTMLElement>(fieldSelector) ??
			node.querySelector<HTMLElement>(focusableSelector)
		target?.focus()
	})

	const trapFocus = (event: KeyboardEvent) => {
		if (event.key !== "Tab") {
			return
		}
		const focusable = Array.from(
			node.querySelectorAll<HTMLElement>(focusableSelector),
		).filter((element) => element.getClientRects().length > 0)
		if (focusable.length === 0) {
			event.preventDefault()
			return
		}

		const first = focusable[0],
			last = focusable.at(-1)
		if (event.shiftKey && document.activeElement === first) {
			event.preventDefault()
			last?.focus()
		} else if (!event.shiftKey && document.activeElement === last) {
			event.preventDefault()
			first?.focus()
		}
	}

	node.addEventListener("keydown", trapFocus)
	return {
		destroy: () => {
			node.removeEventListener("keydown", trapFocus)
			if (previous?.isConnected) {
				previous.focus()
			}
		},
	}
}

function cubicBezier(x1: number, y1: number, x2: number, y2: number): EasingFunction {
	const sample = (a: number, b: number, t: number) =>
			((1 - 3 * b + 3 * a) * t + (3 * b - 6 * a)) * t * t + 3 * a * t,
		derivative = (a: number, b: number, t: number) =>
			3 * (1 - 3 * b + 3 * a) * t * t + 2 * (3 * b - 6 * a) * t + 3 * a

	return (x) => {
		let t = x
		for (let index = 0; index < 5; index += 1) {
			const slope = derivative(x1, x2, t)
			if (Math.abs(slope) < 1e-6) {
				break
			}
			t -= (sample(x1, x2, t) - x) / slope
		}
		return sample(y1, y2, Math.min(1, Math.max(0, t)))
	}
}
