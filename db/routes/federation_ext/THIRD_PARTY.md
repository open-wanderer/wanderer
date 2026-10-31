# Third-party files in federation_ext

Files here are copied byte for byte from the published npm package, never
modified. The page loads nothing from a third-party host.

## Alpine.js CSP build

- Source: npm `@alpinejs/csp`, version 3.17.4, file `package/dist/cdn.min.js`
- Repository: https://github.com/alpinejs/alpine (`packages/csp`), tag `v3.17.4`, gitHead `20ee0e923b3f34e6d6467a31d1b97dd43c85a41d`
- File: `vendor/alpine-csp-3.17.4.min.js`, 72198 bytes, copied unmodified
- sha256: `0d18d7f8d7910e2e0212f0f056b12f50bebc3abb7d88d2f7c7cb4c336fe4519a`
- npm dist.integrity: `sha512-SlRXmqO6kYhnxlg+99etmuzJtE9Lk4QbKjBHqerXzaMflJqoJXdz/SI3IvHJGZ/vRVyC3bR0SSBz40oY7goBeg==`
- The file contains no `new Function` and no `AsyncFunction`, which is why it runs without the unsafe-eval CSP keyword.

Provenance checked by hand on 2026-10-31 before the file was vendored: the npm
maintainer is the same as for `alpinejs` (calebporzio); the package was published
by the GitHub Actions trusted publisher (OIDC); its gitHead equals the GitHub tag
`v3.17.4` and the gitHead of `alpinejs@3.17.4`; the SLSA provenance names
`https://github.com/alpinejs/alpine` at `refs/tags/v3.17.4` built by
`.github/workflows/release.yml` on a GitHub-hosted runner, and its subject sha512
equals the tarball integrity above; `npm audit signatures` verified the registry
signature and attestation. A human approved vendoring this version.

### License (MIT)

The npm tarball carries no license file; the text below is `LICENSE.md` of
alpinejs/alpine at tag `v3.17.4`.

```text
MIT License

Copyright © 2019-2025 Caleb Porzio and contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## IBM Plex Sans (latin subset, weights 400, 500, 600)

- Source: npm `@fontsource/ibm-plex-sans`, version 5.3.0
- npm dist.integrity: `sha512-CbE4CbbEEZJX860XyUiRpsksXIQR8Rp2XDva2VO53NJox9tVNtusrysd2x5YkUEY3ErQ66W1IiiQL8/wihhw5w==`
- License: SIL Open Font License 1.1, text in `fonts/OFL.txt` (the package's LICENSE file)

| File | sha256 |
| --- | --- |
| `fonts/ibm-plex-sans-latin-400-normal.woff2` | `3b646991d30055a93a4ecc499713d4347953a74a947ecab435ab72070cbdab0e` |
| `fonts/ibm-plex-sans-latin-500-normal.woff2` | `0717336fb31fcdcde4b8deb3675bb4a0f7f6d484864afcd6751ac29975962203` |
| `fonts/ibm-plex-sans-latin-600-normal.woff2` | `8960851d691c054ed38e259bdcf1a6190d157b4203ed5bb32c632a863fb8ec2f` |
| `fonts/OFL.txt` | `d0283623ef57e722fd0eb688a8041589670c608ab780cd3612d06ba6f153d3fd` |

## Remix Icon (not vendored)

The icon font is not copied here. PocketBase's embedded admin UI already serves
it from the same origin at `/_/fonts/remixicon/remixicon.woff2`, and
`federation.css` declares its own `@font-face` for that URL plus the 14
codepoints the page uses. `TestFederationIconFontMatchesPocketBase` fails if a
PocketBase upgrade moves the font file or changes any of those codepoints.
