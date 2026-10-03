# Pre-development overrides

The project has not had its first release. Until it does, the rules in this file override every other document. The other documents describe the release design.

Delete this file and the "Pre-development" section of `AGENTS.md` at the first release.

## Compatibility

- Do not preserve backward compatibility. Change APIs, internal interfaces, Wails bindings, database schemas, and file formats directly.
- Remove obsolete code, APIs, and abstractions. Do not keep them for compatibility.
- Do not write adapters, deprecation paths, legacy branches, or compatibility layers.
- Write the clean target design. Do not write transitional code.

## Persisted formats

These rules override the migration and decoder rules in [ARCHITECTURE.md](./ARCHITECTURE.md#persisted-formats).

- Treat `settings.json`, `vault.db`, `vault.key`, and `autounlock.key` as disposable.
- Change the version 1 definition of a format in place. Do not increase a format version.
- Do not write migrations or decoders for older versions. Each list of migrations stays empty, and each list of decoders holds only the version 1 decoder.
- Keep the version scaffolding: version fields, version constants, migration and decoder lists, the upgrade code paths, and the compile-time checks. They are part of the release design.
- After an incompatible format change, delete the local development data so the application recreates it.
