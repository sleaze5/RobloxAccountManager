/// <reference types="svelte" />
/// <reference types="vite/client" />

declare const APP_VERSION: string

type FrontendDependency = { name: string; version: string }

declare const FRONTEND_DEPENDENCIES: {
	runtime: FrontendDependency[]
	build: FrontendDependency[]
}
