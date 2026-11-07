interface PasswordRules {
	length: boolean
	byteLimit: boolean
	uppercase: boolean
	lowercase: boolean
	digit: boolean
	special: boolean
}

function checkPassword(password: string): PasswordRules {
	const characters = Array.from(password)
	return {
		byteLimit: new TextEncoder().encode(password).length <= 1024,
		digit: /[0-9]/.test(password),
		length: characters.length >= 8 && characters.length <= 128,
		lowercase: /[a-z]/.test(password),
		special: characters.some((character) => !/[A-Za-z0-9]/.test(character)),
		uppercase: /[A-Z]/.test(password),
	}
}

export function passwordValid(password: string): boolean {
	return Object.values(checkPassword(password)).every(Boolean)
}

export function passwordRequirements(password: string): [boolean, string][] {
	const rules = checkPassword(password)
	return [
		[rules.length, "8-128 characters long"],
		[rules.uppercase, "At least one upper-case letter"],
		[rules.lowercase, "At least one lower-case letter"],
		[rules.digit, "At least one number"],
		[rules.special, "At least one special character"],
	]
}
