// run starts a command in cwd, shows its errors, and returns its output. It
// throws when the command fails.
export async function run(cwd: string, ...command: string[]): Promise<string> {
	const child = Bun.spawn(command, { cwd, stdout: "pipe", stderr: "inherit" })
	const [output, code] = await Promise.all([
		new Response(child.stdout).text(),
		child.exited,
	])
	if (code !== 0) {
		throw new Error(`${command.join(" ")} failed with exit code ${code}`)
	}
	return output
}
