# dis 🎥

[![GitHub
license](https://img.shields.io/github/license/4evy/dis)](https://github.com/4evy/dis/blob/master/LICENSE)
[![GitHub
release](https://img.shields.io/github/release/4evy/dis)](https://github.com/4evy/dis/releases)
[![GitHub
issues](https://img.shields.io/github/issues/4evy/dis)](https://github.com/4evy/dis/issues)

![The dis command-line help screen](/.github/assets/dis_help.png)

`dis` is a CLI and TUI for downloading, trimming, and compressing videos. It
uses yt-dlp for downloads and FFmpeg for video processing.

## Install

Install [FFmpeg](https://ffmpeg.org/download.html) and
[yt-dlp](https://github.com/yt-dlp/yt-dlp) first. GIF export also requires
[gifski](https://github.com/ImageOptim/gifski).

Download a binary, deb package, or RPM package from [GitHub
Releases](https://github.com/4evy/dis/releases).

### Nix profile

``` sh
nix profile install github:4evy/dis
```

### Nix flake

Add `dis` to your flake inputs:

``` nix
inputs.dis.url = "github:4evy/dis";
```

Use `dis.packages.${system}.default` wherever you define the packages for a
system.

## Build

Build with Go:

``` sh
go build -o dis .
```

Or build with Nix:

``` sh
nix build
```

Run the local checks with `just`:

``` sh
just test
just lint
just test-packages
```

## Browser cookies

For web URLs, `dis` automatically finds Chrome, Chromium, Brave, Edge, Helium,
and Firefox profiles. It asks the installed yt-dlp to validate each browser
session against the requested URL, uses the first working session, and falls
back to no cookies when none work. This works with current and future yt-dlp
extractors without maintaining a separate site list in `dis`.

Firefox containers are treated as separate sessions. `dis` exports each
session to an owner-only temporary cookie jar, reports the profile it selects,
and removes every jar when the command exits.

Use `--cookies-from-browser` to override automatic selection or to load a
specific profile:

``` sh
dis --cookies-from-browser helium:~/.config/net.imput.helium/Default URL
dis --cookies-from-browser firefox:default-release URL
```

Linux Secret Service and KWallet, macOS Keychain, and Windows DPAPI are
handled automatically, so don't add yt-dlp's `+KEYRING` suffix. Helium's
distinct macOS Keychain entry is supported too.

You can also set `cookies_from_browser` in `~/.config/dis/config.toml`.
Current Chrome and Edge releases on Windows may use `v20` app-bound
encryption, which third-party processes can't decrypt. Use Firefox on Windows
when that applies.

## Contribute

Open a [GitHub issue](https://github.com/4evy/dis/issues) to report a bug or
request a feature.

## License

`dis` is available under the [MIT License](LICENSE).
