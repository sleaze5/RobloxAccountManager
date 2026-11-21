// A programming error, such as a TypeError from a missing value or a Wails
// binding that rejects its arguments, carries developer text that does not
// help the user, so it shows the fallback instead.
const programmingErrorTypes = [TypeError, ReferenceError, RangeError, SyntaxError]

export function errorMessage(error: unknown, fallback: string): string {
	if (
		!(error instanceof Error) ||
		programmingErrorTypes.some((errorType) => error instanceof errorType)
	) {
		return fallback
	}
	return error.message.trim() ? error.message : fallback
}
