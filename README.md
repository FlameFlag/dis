# dis 🎥

[![GitHub license](https://img.shields.io/github/license/FlameFlag/dis)](https://github.com/FlameFlag/dis/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/release/FlameFlag/dis)](https://github.com/FlameFlag/dis/releases)
[![GitHub issues](https://img.shields.io/github/issues/FlameFlag/dis)](https://github.com/FlameFlag/dis/issues)

![dis_help](/.github/assets/dis_help.png)

**dis 🎥** is a small and simple CLI and TUI tool designed to download, trim and compress videos for any website

## Building

### Go

All you need is Go installed. Run:

```bash
go build -o dis .
```

### Nix

Simply run `nix build .#default`

Alternatively you can also invoke a `nix shell github:FlameFlag/dis` with the
required packages to build **dis** 🎥

## Installation

To install **dis** 🎥, you need to have [FFmpeg](https://ffmpeg.org/download.html) and [YT-DLP](https://github.com/yt-dlp/yt-dlp) installed on your system. You can download them from their official websites or use your package manager of choice.

You can then download the latest release of **dis** 🎥 from the [Releases](https://github.com/FlameFlag/dis/releases) tab on GitHub. Alternatively, you can clone this repository and build the project yourself using `go build`.

### If you're using Nix

#### Using `nix profile`

```bash
nix profile install github:FlameFlag/dis
```

#### Using flakes

You will need to add **dis** 🎥 to your inputs and pass it down your outputs

```nix
{
  # ...

  inputs = {
    # ...
    dis.url = "github:FlameFlag/dis";
    dis.inputs.nixpkgs.follows = "nixpkgs";
    # ...
  };

  outputs = {
    # ...
    dis,
    # ...
  }
}
```

After that in whichever `.nix` file is responsible for your packages you will need to add **dis** 🎥

Example:

```nix
{ pkgs, dis, ... }: {
  environment = {
    # ...
    systemPackages = builtins.attrValues {
        # ...
        dis = dis.packages.${pkgs.system}.default;
        # ...
    };
  };
}
```

## Contributing

If you want to contribute to **dis** 🎥, you are welcome to do so. You can
report issues, request features, or submit pull requests on GitHub.

## License

**dis** is licensed under the
[MIT](https://github.com/FlameFlag/dis/blob/master/LICENSE).
