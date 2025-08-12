using System.ComponentModel;
using System.Diagnostics;
using System.Runtime.InteropServices;
using dis.Features.Common;
using dis.Features.Conversion;
using dis.Features.Conversion.Models;
using dis.Features.Download.Models;
using dis.Features.Download.Models.Interfaces;
using dis.Features.TrimSlider;
using Microsoft.AspNetCore.StaticFiles;
using Serilog;
using Spectre.Console;
using Spectre.Console.Cli;
using Xabe.FFmpeg;
using YoutubeDLSharp;
using YoutubeDLSharp.Metadata;

namespace dis;

public sealed class RootCommand(
    FileExtensionContentTypeProvider type,
    ILogger logger,
    Globals globals,
    VideoCodecs videoCodecs,
    ValidResolutions validResolutions,
    IDownloader downloader,
    Converter converter)
    : AsyncCommand<Settings>
{
    private static readonly string[] VersionArgs = ["-v", "--version"];

    private void ValidateInputs(IEnumerable<string> inputs)
    {
        if (VersionArgs.Any(Environment.GetCommandLineArgs().Contains))
            return;

        foreach (var input in inputs)
        {
            var isPath = File.Exists(input);
            var isUrl = Uri.IsWellFormedUriString(input, UriKind.RelativeOrAbsolute);
            switch (isPath)
            {
                case false when isUrl is false:
                    {
                        ValidationResult.Error($"Invalid input file or link: {input}");
                        return;
                    }
                case true:
                    {
                        if (type.TryGetContentType(input, out var contentType) is false) return;
                        if (contentType.Contains("video") || contentType.Contains("audio")) return;
                        break;
                    }
            }
        }
    }

    private static void ValidateOutput(string? output)
    {
        if (string.IsNullOrEmpty(output)) output = Environment.CurrentDirectory;
        if (Directory.Exists(output) is false)
            ValidationResult.Error("Output directory does not exist");
    }

    private static void ValidateCrf(int crf)
    {
        const int min = 6;
        const int minRecommended = 22;
        const int max = 63;
        const int maxRecommended = 38;

        var settingsType = typeof(Settings);
        // Use reflection to get the Crf property in the Settings class.
        var crfProperty = settingsType.GetProperty(nameof(Settings.Crf));

        // Get the DefaultValueAttribute assigned to the Crf property.
        var defaultValueAttribute = crfProperty?
            .GetCustomAttributes(typeof(DefaultValueAttribute), false)
            .FirstOrDefault() as DefaultValueAttribute;

        // Get the value from the DefaultValueAttribute.
        var defaultValue = (int)defaultValueAttribute?.Value!;

        /*
         * Use pattern matching with a switch expression to check if the 'crf' value is valid.
         * If 'crf' is less than 0 or greater than the maximum, it is invalid, so 'validCrf' will be set to false.
         * If 'crf' is between the minimum and maximum, inclusive, 'crf' is valid, and 'validCrf' will be set to true.
         */
        var validCrf = crf switch
        {
            < 0 => false,
            >= min and <= max => true,
            _ => false
        };

        if (validCrf is false)
            ValidationResult.Error($"CRF value must be between {min} and {max} (Avoid values below {defaultValue})");
        else switch (crf)
            {
                case < minRecommended:
                    AnsiConsole.MarkupLine($"[yellow]CRF values below {minRecommended} are not recommended[/]");
                    break;
                case > maxRecommended:
                    AnsiConsole.MarkupLine($"[yellow]CRF values above {maxRecommended} are not recommended[/]");
                    break;
            }
    }

    private static void ValidateAudioBitrate(int? audioBitrate)
    {
        if (audioBitrate is null) return;

        var audioBitrateRange = audioBitrate switch
        {
            < 128 => false,
            > 192 => false,
            _ => true
        };
        if (audioBitrateRange is false)
            AnsiConsole.MarkupLine("[yellow]Audio bitrate values below 128 or above 192 are not recommended[/]");
        if (audioBitrate % 2 != 0)
            ValidationResult.Error("Audio bitrate must be a multiple of 2");
    }

    private void ValidateResolution(string? resolution)
    {
        var hasResolution = resolution is not null;
        if (hasResolution is false) return;

        var validResolution = validResolutions.Resolutions
            .Any(res => res.ToString().Equals($"{resolution}p", StringComparison.InvariantCultureIgnoreCase));
        if (validResolution is false)
            ValidationResult.Error("Invalid resolution");
    }

    private void ValidateVideoCodec(string? videoCodec)
    {
        var hasVideoCodec = videoCodec is not null;
        if (!hasVideoCodec) return;

        var validVideoCodec = videoCodecs.Codecs
            .Any(codec => codec.ToString()
                .Equals(videoCodec,
                StringComparison.InvariantCultureIgnoreCase));
        if (validVideoCodec is false)
            ValidationResult.Error("Invalid video codec");
    }

    public override ValidationResult Validate(CommandContext context, Settings settings)
    {
        ValidateInputs(settings.Input);
        ValidateOutput(settings.Output);
        ValidateCrf(settings.Crf);
        ValidateAudioBitrate(settings.AudioBitrate);
        ValidateResolution(settings.Resolution);
        ValidateVideoCodec(settings.VideoCodec);
        ValidationResult.Success();
        return base.Validate(context, settings);
    }

    public override async Task<int> ExecuteAsync(CommandContext context, Settings settings)
    {
        // This is a hacky way to check for the version, but the library doesn't really have a better way. So, we have to do what we have to do.
        if (VersionArgs.Any(Environment.GetCommandLineArgs().Contains))
        {
            AnsiConsole.MarkupLine(typeof(RootCommand).Assembly.GetName().Version!.ToString(3));
            return 0;
        }

        // Patch up output directory to current dir if it's empty
        if (string.IsNullOrEmpty(settings.Output)) settings.Output = Environment.CurrentDirectory;

        var links = settings.Input
            .Where(video =>
                Uri.IsWellFormedUriString(video, UriKind.RelativeOrAbsolute) &&
                File.Exists(video) is false)
            .Select(video => new Uri(video));

        var files = settings.Input
            .Where(File.Exists);

        List<string> paths = [];
        TrimSettings? downloadTrimSettings = null;
        TrimSettings? ffmpegTrimSettings = null;
        Dictionary<Uri, RunResult<VideoData>> runResults = new();

        if (await CheckForFFmpegAndYtDlp() is false)
            return 1;

        // Get trim settings once if trimming is enabled for URLs
        var linksList = links.ToList();
        if (settings.Trim && linksList.Count != 0)
        {
            // Pre-fetch metadata for all videos
            foreach (var link in linksList)
            {
                DownloadOptions downloadOptions = new(link, settings, null);
                var runResult = await downloader.FetchMetadata(downloadOptions);
                if (runResult != null) runResults[link] = runResult;
            }

            // Use the first video with valid duration for trim settings
            foreach (var result in runResults.Values)
            {
                var duration = TimeSpan.FromSeconds(result.Data.Duration ?? 0);
                if (duration <= TimeSpan.Zero) continue;
                var slider = new TrimmingSlider(duration);
                var trimResult = slider.ShowSlider();
                if (string.IsNullOrEmpty(trimResult))  // If canceled, exit immediately
                    return 0;

                var parts = trimResult.Split('-');
                if (parts.Length != 2 ||
                    !double.TryParse(parts[0], out var start) ||
                    !double.TryParse(parts[1], out var end)) continue;
                downloadTrimSettings = new TrimSettings(start, end - start);
                break;
            }
        }

        await Download(linksList, paths, settings, downloadTrimSettings, runResults);

        // Get trim settings for local files if trimming is enabled
        var filesList = files.ToList();
        if (settings.Trim && filesList.Count != 0)
        {
            var mediaInfo = await FFmpeg.GetMediaInfo(filesList.First());

            if (mediaInfo.Streams.FirstOrDefault(s => s is IVideoStream) is IVideoStream videoStream)
            {
                var slider = new TrimmingSlider(videoStream.Duration);
                var trimResult = slider.ShowSlider();
                if (string.IsNullOrEmpty(trimResult))  // If cancelled, exit immediately
                    return 0;

                var parts = trimResult.Split('-');
                if (parts.Length == 2 &&
                    double.TryParse(parts[0], out var start) &&
                    double.TryParse(parts[1], out var end))
                {
                    ffmpegTrimSettings = new TrimSettings(start, end - start);
                }
            }
        }

        paths.AddRange(filesList);

        // Only convert local files here - downloaded files are already converted with their metadata
        var localFiles = filesList.Select(p => (p, (RunResult<VideoData>?)null));
        await Convert(localFiles, settings, ffmpegTrimSettings);

        // Cleanup temp directories after all operations are complete
        var hasAny = globals.TempDir.Count is not 0;
        if (hasAny)
        {
            globals.TempDir.ForEach(d =>
            {
                Directory.Delete(d, true);
                AnsiConsole.MarkupLine($"Deleted temp dir: [red]{d}[/]");
            });
        }

        return 0;
    }

    private static async Task<bool> CheckForFFmpegAndYtDlp()
    {
        try
        {
            var cmd = RuntimeInformation.IsOSPlatform(OSPlatform.Windows) ? "where" : "which";
            var ffmpegPath = await GetCommandPath(cmd, "ffmpeg");
            var ytDlpPath = await GetCommandPath(cmd, "yt-dlp");

            if (string.IsNullOrWhiteSpace(ffmpegPath))
            {
                AnsiConsole.WriteLine("FFmpeg not found in PATH. Please install FFmpeg and add it to PATH.");
                return false;
            }

            if (!string.IsNullOrWhiteSpace(ytDlpPath)) return true;
            AnsiConsole.WriteLine("yt-dlp not found in PATH. Please install yt-dlp and add it to PATH.");
            return false;

        }
        catch (Exception ex)
        {
            AnsiConsole.WriteLine($"An error occurred: {ex.Message}");
            return false;
        }
    }

    private static async Task<string> GetCommandPath(string cmd, string commandName)
    {
        ProcessStartInfo processInfo = new(cmd, commandName)
        {
            RedirectStandardOutput = true,
            UseShellExecute = false,
            CreateNoWindow = true
        };

        using var process = Process.Start(processInfo);
        var commandPath = await process?.StandardOutput.ReadToEndAsync()!;
        await process.WaitForExitAsync();

        return commandPath;
    }

    private async Task Download(
        IEnumerable<Uri> links, 
        List<string> videos, 
        Settings settings, 
        TrimSettings? trimSettings,
        Dictionary<Uri, RunResult<VideoData>> videoMetadata)
    {
        var list = links.ToList();
        if (list.Count is 0)
            return;

        // Track downloaded video paths and their metadata
        var pathToMetadata = new Dictionary<string, RunResult<VideoData>>();

        foreach (var link in list)
        {
            var downloadOptions = new DownloadOptions(link, settings, trimSettings);

            if (!videoMetadata.ContainsKey(link))
            {
                var metadata = await downloader.FetchMetadata(downloadOptions);
                if (metadata != null)
                {
                    videoMetadata[link] = metadata;
                }
            }

            var result = await downloader.DownloadTask(downloadOptions, videoMetadata.GetValueOrDefault(link));

            if (result.OutPath is null)
                logger.Error("There was an error downloading the video");
            else
            {
                videos.Add(result.OutPath);
                if (result.fetchResult != null)
                {
                    pathToMetadata[result.OutPath] = result.fetchResult;
                }
            }
        }

        foreach (var path in videos)
            AnsiConsole.MarkupLine($"Downloaded video to: [green]{path}[/]");

        // Convert videos with their metadata
        await Convert(videos.Select(path => (path, pathToMetadata.GetValueOrDefault(path))), settings, null);
    }

    private async Task Convert(IEnumerable<(string path, RunResult<VideoData>?)> videos, Settings settings, TrimSettings? trimSettings)
    {
        foreach (var (path, fetchResult) in videos)
        {
            try
            {
                await converter.ConvertVideo(path, fetchResult ?? null, settings, trimSettings);
            }
            catch (Exception ex)
            {
                logger.Error(ex, "Failed to convert video: {Path}", path);
            }
        }
    }
}
