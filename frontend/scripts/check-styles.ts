import { readdir, readFile } from "node:fs/promises"
import { join, relative, resolve } from "node:path"
import { __unstable__loadDesignSystem } from "@tailwindcss/node"
import { Scanner } from "@tailwindcss/oxide"

type DesignSystem = Awaited<ReturnType<typeof __unstable__loadDesignSystem>>
type Candidate = ReturnType<DesignSystem["parseCandidate"]>[number]
type Problem = { file: string; line: number; text: string; reason: string }

const frontend = resolve(import.meta.dir, "..")
const source = join(frontend, "src")
const styles = join(source, "styles")
const stylesheet = join(styles, "app.css")

const legacyStyleBlocks = new Set([
	"lib/dialogs/LinuxClientDialog.svelte",
	"lib/dialogs/RobloxPlayerPathDialog.svelte",
	"lib/dialogs/RobloxProcessesDialog.svelte",
	"lib/dialogs/SavedPasswordDialog.svelte",
	"lib/layout/LaunchDrawer.svelte",
	"lib/settings/RobloxSettings.svelte",
	"lib/shared/SensitiveValue.svelte",
	"lib/shared/Timestamp.svelte",
])

const spacingSteps = new Set([
	"0",
	"0.5",
	"1",
	"1.5",
	"2",
	"2.5",
	"3",
	"3.5",
	"4",
	"5",
	"6",
	"7",
	"8",
	"9",
	"10",
	"11",
	"12",
	"14",
	"16",
	"20",
	"24",
	"28",
	"32",
	"36",
	"40",
	"44",
	"48",
	"52",
	"56",
	"60",
	"64",
	"72",
	"80",
	"96",
])
const spacingRoots = new Set([
	"p",
	"px",
	"py",
	"pt",
	"pr",
	"pb",
	"pl",
	"ps",
	"pe",
	"m",
	"mx",
	"my",
	"mt",
	"mr",
	"mb",
	"ml",
	"ms",
	"me",
	"gap",
	"gap-x",
	"gap-y",
	"space-x",
	"space-y",
	"w",
	"min-w",
	"max-w",
	"h",
	"min-h",
	"max-h",
	"size",
	"basis",
	"inset",
	"inset-x",
	"inset-y",
	"top",
	"right",
	"bottom",
	"left",
	"start",
	"end",
	"translate",
	"translate-x",
	"translate-y",
	"scroll-m",
	"scroll-mx",
	"scroll-my",
	"scroll-mt",
	"scroll-mr",
	"scroll-mb",
	"scroll-ml",
	"scroll-p",
	"scroll-px",
	"scroll-py",
	"scroll-pt",
	"scroll-pr",
	"scroll-pb",
	"scroll-pl",
	"indent",
	"leading",
	"border-spacing",
	"border-spacing-x",
	"border-spacing-y",
])
const allowedSteps: Record<string, Set<string>> = {
	z: new Set(["0", "10", "20", "30", "40", "50", "auto"]),
	duration: new Set(["0", "75", "100", "150"]),
	delay: new Set(["0", "75", "100", "150"]),
	scale: new Set(["0", "50", "75", "90", "95", "100", "105", "110"]),
	rotate: new Set(["0", "1", "2", "3", "6", "12", "45", "90", "180"]),
}
for (const root of [
	"border",
	"border-x",
	"border-y",
	"border-t",
	"border-r",
	"border-b",
	"border-l",
	"border-s",
	"border-e",
	"outline",
	"outline-offset",
	"ring",
	"ring-offset",
	"divide-x",
	"divide-y",
]) {
	allowedSteps[root] = new Set(["0", "1", "2", "4", "8"])
}
for (const root of ["scale-x", "scale-y"]) {
	allowedSteps[root] = allowedSteps.scale!
}
const fractionDenominators = new Set(["2", "3", "4", "5", "6", "12"])
const colorRoots = new Set([
	"bg",
	"text",
	"border",
	"border-x",
	"border-y",
	"border-t",
	"border-r",
	"border-b",
	"border-l",
	"border-s",
	"border-e",
	"outline",
	"ring",
	"ring-offset",
	"fill",
	"stroke",
	"divide",
	"decoration",
	"accent",
	"caret",
	"placeholder",
	"shadow",
	"inset-shadow",
	"from",
	"via",
	"to",
])
const textSizes = new Set(["2xs", "xs", "sm", "base", "lg"])
const bannedUtilities = new Map([
	["transition", "List the transitioned properties, such as transition-colors."],
	["transition-all", "List the transitioned properties, such as transition-colors."],
	["animate-ping", "Decorative loops are not allowed."],
	["animate-pulse", "Decorative loops are not allowed."],
	["animate-bounce", "Decorative loops are not allowed."],
])
const arbitraryVariantRoots = new Set([
	"data",
	"aria",
	"group-data",
	"group-aria",
	"peer-data",
	"peer-aria",
])

const themeNamespace =
	/^var\(--(color|font|text|tracking|leading|spacing|radius|shadow|inset-shadow|drop-shadow|blur|ease|animate|container|breakpoint|perspective|aspect|default)\b/

function isStep(value: string): boolean {
	const number = Number(value)
	return Number.isInteger(number) && number >= 0 && number <= 100 && number % 5 === 0
}

function variantProblem(candidate: Candidate): string | null {
	for (const variant of candidate.variants) {
		if (variant.kind === "arbitrary") {
			return "Arbitrary variants are not allowed. Use a named variant or a data attribute."
		}
		if (
			variant.kind === "functional" &&
			variant.value?.kind === "arbitrary" &&
			!arbitraryVariantRoots.has(variant.root)
		) {
			return `Arbitrary values are only allowed in data and aria variants, not in ${variant.root}-[…].`
		}
	}
	return null
}

function modifierProblem(root: string, value: string, fraction: string): string | null {
	const denominator = fraction.split("/")[1] ?? ""
	if (root === "text" && textSizes.has(value)) {
		return "Do not override the line height. Use a leading-* utility."
	}
	if (colorRoots.has(root)) {
		return isStep(denominator) ? null : "Use an opacity modifier in steps of 5."
	}
	return fractionDenominators.has(denominator) ? null : "Use a standard fraction."
}

function valueProblem(root: string, value: string): string | null {
	if (!/^\d/.test(value)) return null
	if (spacingRoots.has(root) && !spacingSteps.has(value)) {
		return `${value} is not on the spacing scale (${[...spacingSteps].join(", ")}).`
	}
	if (root === "opacity" && !isStep(value)) return "Use opacity in steps of 5."
	const steps = allowedSteps[root]
	return steps && !steps.has(value) ? `Use one of ${[...steps].join(", ")}.` : null
}

function runtimeProblem(value: string): string | null {
	if (!/^var\(--[a-z][a-z0-9-]*\)$/.test(value)) {
		return "Read only a runtime variable, such as w-(--sidebar-width)."
	}
	return themeNamespace.test(value)
		? "Use the theme utility instead of reading a theme variable."
		: null
}

function utilityProblem(candidate: Candidate, text: string): string | null {
	const banned = bannedUtilities.get(
		text.split(":").at(-1)?.replaceAll("!", "") ?? "",
	)
	if (banned) return banned
	if (candidate.kind === "arbitrary") return "Arbitrary properties are not allowed."
	if (candidate.kind !== "functional" || !candidate.value) return null
	if (candidate.value.kind === "arbitrary" && text.includes("-(--")) {
		return runtimeProblem(candidate.value.value)
	}
	if (
		candidate.value.kind === "arbitrary" ||
		candidate.modifier?.kind === "arbitrary"
	) {
		return "Arbitrary values are not allowed. Use a theme token or the spacing scale."
	}
	const root = candidate.root.replace(/^-/, "")
	const { fraction, value } = candidate.value
	return fraction ? modifierProblem(root, value, fraction) : valueProblem(root, value)
}

async function files(directory: string): Promise<string[]> {
	const entries = await readdir(directory, { recursive: true, withFileTypes: true })
	return entries
		.filter((entry) => entry.isFile())
		.map((entry) => join(entry.parentPath, entry.name))
}

function withoutStyleBlocks(content: string): string {
	return content.replaceAll(/<style[\s\S]*?<\/style>/g, (block) =>
		block.replaceAll(/[^\n]/g, " "),
	)
}

function lineOf(content: string, index: number): number {
	return content.slice(0, index).split("\n").length
}

function checkClasses(
	sources: Map<string, string>,
	designSystem: DesignSystem,
	problems: Problem[],
): void {
	const scanner = new Scanner({ sources: [] })
	const valid = new Map<string, boolean>()
	const isValid = (text: string) => {
		if (!valid.has(text)) {
			valid.set(text, designSystem.candidatesToCss([text])[0] !== null)
		}
		return valid.get(text)
	}
	for (const [path, original] of sources) {
		const content = withoutStyleBlocks(original)
		const extension = path.split(".").at(-1) ?? ""
		for (const { candidate: text, position } of scanner.getCandidatesWithPositions({
			content,
			extension,
		})) {
			const candidate = isValid(text)
				? designSystem.parseCandidate(text)[0]
				: null
			const reason =
				candidate &&
				(variantProblem(candidate) ?? utilityProblem(candidate, text))
			if (reason) {
				problems.push({
					file: path,
					line: lineOf(content, position),
					text,
					reason,
				})
			}
		}
	}
}

function checkSource(
	path: string,
	content: string,
	designSystem: DesignSystem,
	problems: Problem[],
): void {
	const name = relative(source, path).replaceAll("\\", "/")
	const report = (index: number, text: string, reason: string) =>
		problems.push({ file: path, line: lineOf(content, index), text, reason })
	if (path.endsWith(".svelte") && !legacyStyleBlocks.has(name)) {
		for (const match of content.matchAll(/<style[\s>]/g)) {
			report(match.index, "<style>", "Style the markup with utilities instead.")
		}
	}
	for (const match of content.matchAll(/\bstyle="([^"]*)"/g)) {
		const literal = (match[1] ?? "").replaceAll(/\{[^}]*\}/g, "")
		if (/\d(px|rem|em|ms|s|%)\b|#[0-9a-f]{3,8}\b|rgba?\(/i.test(literal)) {
			report(match.index, match[0], "Inline styles may only pass runtime values.")
		}
	}
	for (const match of content.matchAll(
		/(?<![\w-])((?:[a-z0-9-]+:)*)(-?[a-z][a-z0-9-]*?)-(\$\{|\{)/g,
	)) {
		const before = content.slice(Math.max(0, match.index - 300), match.index)
		const inClass = match[3] === "${" || /\bclass="[^"]*$/.test(before)
		if (inClass && designSystem.utilities.has(match[2] ?? "", "functional")) {
			report(
				match.index,
				match[0],
				"Write each class name in full, never built from parts.",
			)
		}
	}
}

function checkStylesheet(path: string, content: string, problems: Problem[]): void {
	if (path.endsWith("theme.css") || path.endsWith("base.css")) return
	for (const match of content.matchAll(
		/#[0-9a-f]{3,8}\b|rgba?\([^)]*\)|\b(?!0px\b|1px\b|2px\b)\d+(\.\d+)?(px|ms)\b/gi,
	)) {
		problems.push({
			file: path,
			line: lineOf(content, match.index),
			text: match[0],
			reason: "Use a theme variable or --spacing() instead of a raw value.",
		})
	}
}

const problems: Problem[] = []
const paths = (await files(source)).filter(
	(path) => !path.startsWith(join(styles, "legacy")),
)
const contents = new Map(
	await Promise.all(
		paths.map(async (path) => [path, await readFile(path, "utf8")] as const),
	),
)
const markup = new Map([...contents].filter(([path]) => /\.(svelte|ts)$/.test(path)))
const designSystem = await __unstable__loadDesignSystem(
	await readFile(stylesheet, "utf8"),
	{
		base: styles,
	},
)
checkClasses(markup, designSystem, problems)
for (const [path, content] of contents) {
	if (markup.has(path)) checkSource(path, content, designSystem, problems)
	else if (path.endsWith(".css")) checkStylesheet(path, content, problems)
}

for (const problem of problems) {
	console.error(
		`${relative(frontend, problem.file)}:${problem.line}: ${problem.text}: ${problem.reason}`,
	)
}
if (problems.length > 0) {
	console.error(`\n${problems.length} style problems found.`)
	process.exit(1)
}
console.log(`Style check passed for ${markup.size} files.`)
