# Third-party notices

Roblox Account Manager is built with the third-party works listed below. Each work remains under its own license. [THIRD_PARTY_LICENSES.txt](./THIRD_PARTY_LICENSES.txt) contains their copyright notices and license texts, including those of code that these works bundle.

## Services

### RoValra

Thanks to Valra and the RoValra project for the RoValra API at `apis.rovalra.com`.

Requests to the RoValra API include only place IDs and server IDs. Requests never include account credentials. The application does not include RoValra code or assets.

- Website: <https://www.rovalra.com>
- Terms of use: <https://www.rovalra.com/tou/>

## Fonts

The application bundles unmodified upright variable WOFF2 fonts from Vercel's Geist release `v1.7.2`. Both include weights 100 through 900 and the upstream glyph sets. No italic face, font package, or runtime font request is included.

- Website: <https://vercel.com/font>
- Upstream release: <https://github.com/vercel/geist-font/releases/tag/v1.7.2>

### Geist

- Source file: [`fonts/Geist/webfonts/Geist[wght].woff2`](https://github.com/vercel/geist-font/blob/v1.7.2/fonts/Geist/webfonts/Geist%5Bwght%5D.woff2)
- Bundled file: `frontend/src/assets/fonts/Geist-Variable.woff2`
- SHA-256: `2ffebe993e969069a9789d15164b7715d42491b5835516c5e3b935d5f81b05f1`

### Geist Mono

- Source file: [`fonts/GeistMono/webfonts/GeistMono[wght].woff2`](https://github.com/vercel/geist-font/blob/v1.7.2/fonts/GeistMono/webfonts/GeistMono%5Bwght%5D.woff2)
- Bundled file: `frontend/src/assets/fonts/GeistMono-Variable.woff2`
- SHA-256: `afaacc4c5fbba89d2ebf7a02dc4070208540874592a5504d57175782fe893101`

## Libraries

- [Phosphor Icons](https://phosphoricons.com/)
- [Svelte](https://svelte.dev/)
- [three.js](https://threejs.org/)
- [Wails](https://wails.io/)
- [SQLite](https://sqlite.org/) through [go-sqlite3](https://github.com/ncruces/go-sqlite3)
- [Go](https://go.dev/) and its `golang.org/x` modules
- [go-keyring](https://github.com/zalando/go-keyring), [dbus](https://github.com/godbus/dbus), [go-ole](https://github.com/go-ole/go-ole), and [xdg](https://github.com/adrg/xdg)
- [compress](https://github.com/klauspost/compress), for reading Roblox model files
- [RoAvatar Renderer](https://github.com/steinann/RoAvatar-Renderer) `1.10.3`, for the 3D avatar on the avatar page, with the [Draco](https://github.com/google/draco) mesh decoder, [postprocessing](https://github.com/pmndrs/postprocessing), [N8AO](https://github.com/N8python/n8ao), and [fzstd](https://github.com/101arrowz/fzstd) that it bundles

## RoAvatar Renderer

The 3D avatar is drawn by RoAvatar Renderer, by the RoAvatar authors and steinann, under the GNU General Public License version 3. Its use is why Roblox Account Manager is released under the same license. RoAvatar's name and branding are not part of that license, and the application does not use them.

The renderer downloads the files it draws, such as the avatar rig and the classic body meshes, from Roblox through the signed-in account. The application does not bundle these files.

- Source: <https://github.com/steinann/RoAvatar-Renderer>
- Package: <https://www.npmjs.com/package/roavatar-renderer>
