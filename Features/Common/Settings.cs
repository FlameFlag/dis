using System.ComponentModel;
using Spectre.Console.Cli;

namespace dis.Features.Common;

public sealed class Settings : CommandSettings
{
    [Description("Input url/file")]
    [CommandArgument(0, "<input>")]
    [CommandOption("-i|--input")]
    public required string[] Input { get; init; }

    [Description("Output directory")]
    [CommandArgument(0, "<output>")]
    [CommandOption("-o|--output")]
    public string? Output { get; set; }

    [Description("""
                 Constant Rate Factor (CRF)
                 Higher values mean lower quality
                 Lower values mean higher quality
                 You should use a sane value between 22 and 38
                 """)]
    [CommandOption("-c|--crf")]
    [DefaultValue(25)]
    public int Crf { get; set; }

    [Description("""
                 Resolution to be used for when compressing the video.
                 Available resolutions:
                 144p, 240p, 360p, 480p, 720p, 1080p, 1440p, 2160p
                 """)]
    [CommandOption("-r|--resolution")]
    public string? Resolution { get; set; }

    [Description("Enable video trimming")]
    [CommandOption("-t|--trim")]
    [DefaultValue(false)]
    public bool Trim { get; set; }

    [Description("""
                 Avaliable codecs:
                 h264, libx264 - default
                 vp9, libvpx-vp9 - VP9 (Webm) is recommended for 1080p
                 av1, libaom-av1 - AV1 is recommended for 4K
                 """)]
    [CommandOption("--video-codec")]
    [DefaultValue("libx264")]
    public required string VideoCodec { get; init; }

    [Description("Output audio bitrate (in kbit/s)")]
    [CommandOption("--audio-bitrate")]
    public int? AudioBitrate { get; init; }

    [Description("Use all available threads (for faster compression)")]
    [CommandOption("--multi-thread")]
    [DefaultValue(true)]
    public bool MultiThread { get; set; }

    [Description("The output file name will be random file name")]
    [CommandOption("--random")]
    [DefaultValue(false)]
    public bool RandomFileName { get; init; }

    [Description("Remove all sponsor from YouTube Video")]
    [CommandOption("--sponsor")]
    [DefaultValue(false)]
    public bool SponsorBlock { get; init; }
}
