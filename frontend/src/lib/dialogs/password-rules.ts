export interface PasswordRules {
	length: boolean
	byteLimit: boolean
	uppercase: boolean
	lowercase: boolean
	digit: boolean
	special: boolean
}

export function checkPassword(password: string): PasswordRules {
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
