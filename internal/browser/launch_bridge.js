;(() => {
	if (location.origin !== "https://www.roblox.com" && location.origin !== "https://roblox.com") return

	const send = globalThis.__ramLaunchRoblox
	function handoff(value) {
		if (typeof value !== "string" || !/^roblox-player:/i.test(value)) return false
		send(value)
		return true
	}

	// Roblox's game launcher sets an iframe's src before inserting it. Stop the
	// native navigation synchronously; a MutationObserver would run too late.
	const setAttribute = Element.prototype.setAttribute
	Element.prototype.setAttribute = function (name, value) {
		if (this instanceof HTMLIFrameElement && String(name).toLowerCase() === "src" && handoff(value)) return
		return Reflect.apply(setAttribute, this, [name, value])
	}
	const src = Object.getOwnPropertyDescriptor(HTMLIFrameElement.prototype, "src")
	Object.defineProperty(HTMLIFrameElement.prototype, "src", {
		...src,
		set(value) {
			if (!handoff(value)) Reflect.apply(src.set, this, [value])
		},
	})

	const open = window.open
	window.open = function (url, ...args) {
		if (handoff(url)) return null
		return Reflect.apply(open, this, [url, ...args])
	}
	window.navigation.addEventListener("navigate", (event) => {
		if (event.cancelable && handoff(event.destination.url)) event.preventDefault()
	})
	document.addEventListener(
		"click",
		(event) => {
			const anchor = event.target instanceof Element ? event.target.closest("a[href]") : null
			if (anchor && !event.defaultPrevented && handoff(anchor.href)) {
				event.preventDefault()
				event.stopImmediatePropagation()
			}
		},
		true,
	)
})()
