// CancellableRequest is a backend call that can be cancelled, such as the
// promise that a Wails binding returns.
export interface CancellableRequest<T> extends Promise<T> {
	cancel(): unknown
}
