// The kinds of Luau tokens that the source viewer sets apart. Everything else
// is plain code.
export type LuauTokenKind = "code" | "comment" | "string" | "keyword" | "number"

interface LuauToken {
	kind: LuauTokenKind
	text: string
}

// luauTokenPattern matches, in order of precedence: long comments, line
// comments, long strings, quoted and interpolated strings, keywords, and
// numbers. A long bracket closes with as many equals signs as it opened with.
const luauTokenPattern =
	/(--\[(=*)\[[\s\S]*?\]\2\]|--[^\n]*)|(\[(=*)\[[\s\S]*?\]\4\]|"(?:\\[\s\S]|[^"\\\n])*"|'(?:\\[\s\S]|[^'\\\n])*'|`(?:\\[\s\S]|[^`\\])*`)|\b(and|break|continue|do|else|elseif|end|export|false|for|function|if|in|local|nil|not|or|repeat|return|then|true|type|until|while)\b|(\b0x[\da-fA-F_]+\b|\b\d[\d_]*(?:\.\d+)?(?:[eE][+-]?\d+)?\b)/g

// luauLines splits source code into lines of tokens. A token that spans
// lines, such as a long comment, is split at each line break, so that every
// line keeps its own kind of token.
export function luauLines(source: string): LuauToken[][] {
	const lines: LuauToken[][] = [[]]
	const addText = (kind: LuauTokenKind, text: string): void => {
		text.split("\n").forEach((lineText, index) => {
			if (index > 0) lines.push([])
			if (lineText) lines.at(-1)?.push({ kind, text: lineText })
		})
	}
	const code = source.replaceAll("\r\n", "\n")
	let end = 0
	for (const match of code.matchAll(luauTokenPattern)) {
		addText("code", code.slice(end, match.index))
		const kind: LuauTokenKind = match[1]
			? "comment"
			: match[3]
				? "string"
				: match[5]
					? "keyword"
					: "number"
		addText(kind, match[0])
		end = match.index + match[0].length
	}
	addText("code", code.slice(end))
	return lines
}
