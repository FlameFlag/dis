# dis 🎥

[![GitHub license](https://img.shields.io/github/license/DontEatOreo/dis)](https://github.com/DontEatOreo/dis/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/release/DontEatOreo/dis)](https://github.com/DontEatOreo/dis/releases)
[![GitHub issues](https://img.shields.io/github/issues/DontEatOreo/dis)](https://github.com/DontEatOreo/dis/issues)

![292BEC87](https://github.com/user-attachments/assets/395cbfa1-f86c-403c-a40e-c33ee9200ce3)

**dis** 🎥 is a simple CLI tool to compress videos for platforms like Discord
and likewise. It uses [YT-DLP](https://github.com/yt-dlp/yt-dlp) to download
videos from all sites that `yt-dlp` supports, such as YouTube, Twitter, Reddit,
and more, and [FFmpeg](https://ffmpeg.org/download.html) for compression. It
comes with a rich and easy to use TUI mode for trimming videos

## Nix Dev Environment

**dis** 🎥 comes with a `nix develop` environment that carries all the
dependencies (dotnet, ffmpeg, and yt-dlp), allowing you to work on the project
without having to manually install them. To activate this environment, all you
need to do is:

1. Have `nix` installed
2. Have `flakes` and `nix-command` experimental features enabled
3. Be in the same directory
4. Run `nix develop --impure`

## Building

### .NET

All you need to do is run `dotnet build`, then a `bin/` folder will be created
that will contain all the build artifacts

### Building a Single File Executable

You can build a single file executable which you can run without having dotnet
preinstalled and that you can place anywhere. To do that all you need to do is:

1. `dotnet publish dis.csproj -p:PublishSingleFile=true
-p:IncludeNativeLibrariesForSelfExtract=true --self-contained
true --configuration Release --runtime <rid>`
([KnowRIDs](https://learn.microsoft.com/en-us/dotnet/core/rid-catalog#known-rids))
2. Then you can find the executable at `bin/Release/net8.0/<rid>/`

### Nix

### Simply run `nix build .#default`

Alternatively you can also invoke a `nix shell github:DontEatOreo/dis` with the
required packages to build **dis** 🎥

### Updating dependencies on Nix

1. Run `nix run nixpkgs#dotnetCorePackages.sdk_10_0-bin -- restore --packages out`
2. Run the script `nix-shell -p dotnetCorePackages.sdk_10_0-bin nuget-to-json --run "nuget-to-json out > deps.nix"`

## Installation

To install **dis** 🎥, you need to have
[FFmpeg](https://ffmpeg.org/download.html) and
[YT-DLP](https://github.com/yt-dlp/yt-dlp) installed on your system. You can
download them from their official websites or use your package manager of
choice.

You can then download the latest release of **dis** 🎥 from
the [Releases](https://github.com/DontEatOreo/dis/releases)
tab on GitHub. Alternatively, you can clone this
repository and build the project yourself using [dotnet
publish](https://docs.microsoft.com/en-us/dotnet/core/tools/dotnet-publish)

### If you're using Nix

### Using `nix profile`

```bash
nix profile install github:DontEatOreo/dis
```

### Using flakes

You will need to add **dis** 🎥 to your inputs and pass it down your outputs

```nix
{
  # ...

  inputs = {
    # ...
    dis.url = "github:DontEatOreo/dis";
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

After that in whichever `.nix` file is responsible for your packages you will
need to add **dis** 🎥

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

## Usage

To use **dis** 🎥, you need to provide an input video source using the `-i`
(`--input`) option. The input source can be either a local file or an online URL
(e.g., Discord Server, YouTube, Twitter, Reddit, etc...).

You can also specify various options to customize the output video, such as:

| Option                              | Description                                                                                           | Default Value |
|-------------------------------------|-------------------------------------------------------------------------------------------------------|---------------|
| `-i`, `--input` `<input>`           | Input video source                                                                                    | None          |
| `-o`, `--output` `<output>`         | Output directory for the compressed video                                                             | Current Path  |
| `-c`, `--crf` `<crf>`               | Constant Rate Factor (CRF) Higher values mean lower quality, Lower values mean higher quality         | `25`          |
| `-r`, `--resolution` `<resolution>` | Output video resolution                                                                               | None          |
| `-t`, `--trim`                      | Enable video trimming                                                                                 | `false`       |
| `--video-codec` `<video-codec>`     | Output video codec                                                                                    | None          |
| `--audio-bitrate` `<audio-bitrate>` | Output audio bitrate (in kbit/s) (only divisible by 2)                                                | `128k`        |
| `--multi-thread`                    | Use all available threads (for faster compression)                                                    | true          |
| `--random`                          | Randomize the output filename                                                                         | `false`       |
| `--sponsor`                         | Remove the sponsor segments from the output video using [SponsorBlock](https://sponsor.ajay.app/) API | `False`       |

The CRF is a video compression method that grants users control over the output
video quality, with higher values resulting in lower quality and vice versa. It
is recommended to avoid values below 22 or above 38.

## Here are some examples of how to use **dis** 🎥

### Compressing a video from a website

![1](https://github.com/user-attachments/assets/71afc721-d57d-4a36-99f8-c236a58c2b72)

### Compressing a local file while adjusting its resolution to `480p` and saving it to a designated directory

![2](https://github.com/user-attachments/assets/817a11c3-7be4-4890-a227-da1391f70112)

### Compressing a video from YouTube and change the video codec to `h265`, resolution to `720p`, and audio bitrate to `128`, with a custom CRF value

![3](https://github.com/user-attachments/assets/d08d5424-fa13-4fda-aef1-963d25a2c2aa)

### Compressing a video from YouTube and trimming from 10 seconds to 20 seconds

![4](https://github.com/user-attachments/assets/f648df70-0ce8-49bd-bdfd-4130d11840d3)

## Contributing

If you want to contribute to **dis** 🎥, you are welcome to do so. You can
report issues, request features, or submit pull requests on GitHub.

## License

**dis** is licensed under the
[AGPLv3](https://github.com/DontEatOreo/dis/blob/master/LICENSE).
