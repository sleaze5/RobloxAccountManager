// UnsavedChangesGuard lets the page with unsaved changes ask the user before
// a navigation closes it.
class UnsavedChangesGuard {
	#confirmLeaving: ((leave: () => void) => boolean) | null = null

	// hold registers the page's confirmation, which returns whether it holds the
	// navigation and runs leave once the user confirms. It returns a function
	// that removes the confirmation.
	hold(confirmLeaving: (leave: () => void) => boolean): () => void {
		this.#confirmLeaving = confirmLeaving
		return () => {
			if (this.#confirmLeaving === confirmLeaving) this.#confirmLeaving = null
		}
	}

	// leave navigates now, or once the user confirms discarding the changes.
	leave(navigate: () => void): void {
		if (!this.#confirmLeaving?.(navigate)) navigate()
	}
}

export const unsavedChangesGuard = new UnsavedChangesGuard()
