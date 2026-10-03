# Third-party notices

Roblox Account Manager includes or uses the third-party works listed below. Each work remains under its own license.

## Services

### RoValra

Thanks to Valra and the RoValra project for the RoValra API at `apis.rovalra.com`.

Requests to the RoValra API include only place IDs and server IDs. Requests never include account credentials. The application does not include RoValra code or assets.

- Website: <https://www.rovalra.com>
- Terms of use: <https://www.rovalra.com/tou/>

## Bundled fonts

The application bundles upright variable fonts locally so startup remains offline and portable. No italic face or runtime font request is included.

### Nunito Sans

- Upstream: <https://github.com/googlefonts/nunito>
- Source commit: `8c6a9bb9732545b9ed53f29ec5e1ab0ff53c4e6f`
- Source file: `fonts/variable/Nunito[wght].ttf`
- Bundled file: `frontend/src/assets/fonts/NunitoSans-Variable.woff2`
- Included weight range: 400 through 700
- Copyright: Copyright 2014 The Nunito Project Authors (https://github.com/googlefonts/nunito)
- License: [SIL Open Font License 1.1](#sil-open-font-license-11)

### JetBrains Mono

- Upstream: <https://github.com/JetBrains/JetBrainsMono>
- Source release: `v2.304`
- Source file: `fonts/variable/JetBrainsMono[wght].ttf` from `JetBrainsMono-2.304.zip`
- Bundled file: `frontend/src/assets/fonts/JetBrainsMono-Variable.woff2`
- Included weight range: 400 through 600
- Copyright: Copyright 2020 The JetBrains Mono Project Authors (https://github.com/JetBrains/JetBrainsMono)
- License: [SIL Open Font License 1.1](#sil-open-font-license-11)

### Reproducible subset

The checked-in WOFF2 files were generated with FontTools 4.63.0 and Brotli 1.2.0. First restrict the `wght` axis with `fontTools.varLib.instancer`, then run `fontTools.subset` with:

```text
--flavor=woff2
--unicodes=U+0000-00FF,U+0100-024F,U+1E00-1EFF,U+2000-206F,U+20A0-20CF,U+2100-214F,U+2190-21FF,U+2200-22FF,U+25A0-25FF,U+FEFF,U+FFFD
--layout-features=*
--name-IDs=*
--name-legacy
--name-languages=*
--notdef-glyph
--notdef-outline
--recommended-glyphs
```

Nunito Sans uses `wght=400:700`; JetBrains Mono uses `wght=400:600`.

## License texts

### SIL Open Font License 1.1

This license applies to Nunito Sans and JetBrains Mono. It is also available with a FAQ at <https://openfontlicense.org>.

```text
-----------------------------------------------------------
SIL OPEN FONT LICENSE Version 1.1 - 26 February 2007
-----------------------------------------------------------

PREAMBLE
The goals of the Open Font License (OFL) are to stimulate worldwide
development of collaborative font projects, to support the font creation
efforts of academic and linguistic communities, and to provide a free and
open framework in which fonts may be shared and improved in partnership
with others.

The OFL allows the licensed fonts to be used, studied, modified and
redistributed freely as long as they are not sold by themselves. The
fonts, including any derivative works, can be bundled, embedded,
redistributed and/or sold with any software provided that any reserved
names are not used by derivative works. The fonts and derivatives,
however, cannot be released under any other type of license. The
requirement for fonts to remain under this license does not apply
to any document created using the fonts or their derivatives.

DEFINITIONS
"Font Software" refers to the set of files released by the Copyright
Holder(s) under this license and clearly marked as such. This may
include source files, build scripts and documentation.

"Reserved Font Name" refers to any names specified as such after the
copyright statement(s).

"Original Version" refers to the collection of Font Software components as
distributed by the Copyright Holder(s).

"Modified Version" refers to any derivative made by adding to, deleting,
or substituting -- in part or in whole -- any of the components of the
Original Version, by changing formats or by porting the Font Software to a
new environment.

"Author" refers to any designer, engineer, programmer, technical
writer or other person who contributed to the Font Software.

PERMISSION & CONDITIONS
Permission is hereby granted, free of charge, to any person obtaining
a copy of the Font Software, to use, study, copy, merge, embed, modify,
redistribute, and sell modified and unmodified copies of the Font
Software, subject to the following conditions:

1) Neither the Font Software nor any of its individual components,
in Original or Modified Versions, may be sold by itself.

2) Original or Modified Versions of the Font Software may be bundled,
redistributed and/or sold with any software, provided that each copy
contains the above copyright notice and this license. These can be
included either as stand-alone text files, human-readable headers or
in the appropriate machine-readable metadata fields within text or
binary files as long as those fields can be easily viewed by the user.

3) No Modified Version of the Font Software may use the Reserved Font
Name(s) unless explicit written permission is granted by the corresponding
Copyright Holder. This restriction only applies to the primary font name as
presented to the users.

4) The name(s) of the Copyright Holder(s) or the Author(s) of the Font
Software shall not be used to promote, endorse or advertise any
Modified Version, except to acknowledge the contribution(s) of the
Copyright Holder(s) and the Author(s) or with their explicit written
permission.

5) The Font Software, modified or unmodified, in part or in whole,
must be distributed entirely under this license, and must not be
distributed under any other license. The requirement for fonts to
remain under this license does not apply to any document created
using the Font Software.

TERMINATION
This license becomes null and void if any of the above conditions are
not met.

DISCLAIMER
THE FONT SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO ANY WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT
OF COPYRIGHT, PATENT, TRADEMARK, OR OTHER RIGHT. IN NO EVENT SHALL THE
COPYRIGHT HOLDER BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
INCLUDING ANY GENERAL, SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL
DAMAGES, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF THE USE OR INABILITY TO USE THE FONT SOFTWARE OR FROM
OTHER DEALINGS IN THE FONT SOFTWARE.
```
