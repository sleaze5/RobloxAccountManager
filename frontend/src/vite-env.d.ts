/// <reference types="svelte" />
/// <reference types="vite/client" />

type FrontendDependency = { name: string; version: string }

declare const FRONTEND_DEPENDENCIES: {
	runtime: FrontendDependency[]
	build: FrontendDependency[]
}
