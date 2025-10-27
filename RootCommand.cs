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

    private ValidationResult ValidateInputs(IEnumerable<string> inputs)
    {
        if (IsVersionRequest()) return ValidationResult.Success();

        foreach (var input in inputs)
        {
            var isPath = File.Exists(input);
            var isUrl = Uri.IsWellFormedUriString(input, UriKind.Absolute);

            switch (isPath)
            {
                case false when !isUrl:
                    return ValidationResult.Error($"Invalid input file or link: {input}");
                case true:
                {
                    if (!type.TryGetContentType(input, out var contentType))
                    {
                        logger.Warning("Could not determine content type for file: {Input}", input);
                        continue;
                    }

                    var isMedia = contentType.Contains("video", StringComparison.OrdinalIgnoreCase) ||
                                  contentType.Contains("audio", StringComparison.OrdinalIgnoreCase);
                    if (!isMedia)
                        return ValidationResult.Error($"Input file is not a recognized video/audio type: {input} (Type: {contentType})");

                    break;
                }
            }
        }
        return ValidationResult.Success();
    }

    private static ValidationResult ValidateOutput(string? output)
    {
        if (!string.IsNullOrEmpty(output) && !Directory.Exists(output))
            return ValidationResult.Error($"Output directory does not exist: {output}");
        
        return ValidationResult.Success();
    }

    private static ValidationResult ValidateCrf(int crf)
    {
        const int min = 6;
        const int max = 63;
        const int minRecommended = 22;
        const int maxRecommended = 38;

        switch (crf)
        {
            case < min or > max:
                return ValidationResult.Error($"CRF value must be between {min} and {max} (Recommended: {minRecommended}-{maxRecommended})");
            case < minRecommended:
                AnsiConsole.MarkupLine($"[yellow]Warning: CRF value {crf} is below the recommended minimum of {minRecommended}. This may result in very large files.[/]");
                break;
            case > maxRecommended:
                AnsiConsole.MarkupLine($"[yellow]Warning: CRF value {crf} is above the recommended maximum of {maxRecommended}. This may result in poor quality.[/]");
                break;
        }

        return ValidationResult.Success();
    }

    private static ValidationResult ValidateAudioBitrate(int? audioBitrate)
    {
        switch (audioBitrate)
        {
            case null:
                return ValidationResult.Success();
            case < 128 or > 192:
                AnsiConsole.MarkupLine("[yellow]Warning: Audio bitrate values outside the 128-192 kbps range are not generally recommended.[/]");
                break;
        }

        return audioBitrate % 2 != 0
            ? ValidationResult.Error("Audio bitrate must be a multiple of 2.")
            : ValidationResult.Success();
    }

    private ValidationResult ValidateResolution(string? resolution)
    {
        if (string.IsNullOrEmpty(resolution)) return ValidationResult.Success();
        
        var validResolutionsText = string.Join(", ", validResolutions.Resolutions.Select(r => $"{r}p"));
        var isValid = validResolutions.Resolutions
            .Any(res => res.ToString().Equals(resolution.Replace("p", ""), StringComparison.InvariantCultureIgnoreCase));

        return !isValid
            ? ValidationResult.Error($"Invalid resolution: {resolution}. Valid options are: {validResolutionsText}")
            : ValidationResult.Success();
    }
    
    private ValidationResult ValidateVideoCodec(string? videoCodec)
    {
        if (string.IsNullOrEmpty(videoCodec)) return ValidationResult.Success();

        var validCodecsText = string.Join(", ", videoCodecs.Codecs);
        var isValid = videoCodecs.Codecs
            .Any(codec => codec.ToString().Contains(videoCodec, StringComparison.InvariantCultureIgnoreCase));

        return !isValid
            ? ValidationResult.Error($"Invalid video codec: {videoCodec}. Valid options are: {validCodecsText}")
            : ValidationResult.Success();
    }

    public override ValidationResult Validate(CommandContext context, Settings settings)
    {
        var validationResults = new[]
        {
            ValidateInputs(settings.Input),
            ValidateOutput(settings.Output),
            ValidateCrf(settings.Crf),
            ValidateAudioBitrate(settings.AudioBitrate),
            ValidateResolution(settings.Resolution),
            ValidateVideoCodec(settings.VideoCodec)
        };

        return validationResults.FirstOrDefault(result => !result.Successful) ?? base.Validate(context, settings);
    }
    
    public override async Task<int> ExecuteAsync(CommandContext context, Settings settings, CancellationToken cancellationToken)
    {
        if (IsVersionRequest())
        {
            PrintVersion();
            return 0;
        }

        if (!await CheckDependenciesAsync()) return 1;

        settings.Output ??= Environment.CurrentDirectory;

        var (links, localFiles) = CategorizeInputs(settings.Input);

        if (links.Count == 0 && localFiles.Count == 0)
        {
            logger.Warning("No valid input links or local files were provided.");
            return 1;
        }

        var trimSettings = await GetTrimSettingsAsync(settings, links, localFiles, cancellationToken);
        if (settings.Trim && trimSettings is null)
        {
            logger.Information("Trimming was cancelled by the user.");
            return 0;
        }
        
        var downloadedItems = await DownloadAllAsync(links, settings, trimSettings, cancellationToken);
        var localItems = localFiles.Select(path => new ConversionItem(path, null)).ToList();

        var itemsProcessed = false;

        // Convert downloaded items, which are already trimmed. Pass null for trimSettings.
        var conversionItems = downloadedItems.ToList();
        if (conversionItems.Count != 0)
        {
            await ConvertAllAsync(conversionItems, settings, null, cancellationToken);
            itemsProcessed = true;
        }

        // Convert local files, which need to be trimmed by FFmpeg. Pass the original trimSettings.
        if (localItems.Count != 0)
        {
            await ConvertAllAsync(localItems, settings, trimSettings, cancellationToken);
            itemsProcessed = true;
        }

        if (!itemsProcessed)
        {
            logger.Information("No videos were successfully processed for conversion.");
            return 0;
        }

        CleanupTempDirectories();
        return 0;
    }

    private static bool IsVersionRequest() => VersionArgs.Any(Environment.GetCommandLineArgs().Contains);
    
    private static void PrintVersion() => AnsiConsole.MarkupLine(typeof(RootCommand).Assembly.GetName().Version!.ToString(3));

    private (List<Uri> Uris, List<string> FilePaths) CategorizeInputs(IEnumerable<string> inputs)
    {
        var uris = new List<Uri>();
        var filePaths = new List<string>();
        foreach (var input in inputs)
        {
            if (File.Exists(input))
            {
                filePaths.Add(input);
            }
            else if (Uri.IsWellFormedUriString(input, UriKind.Absolute))
            {
                uris.Add(new Uri(input));
            }
        }
        return (uris, filePaths);
    }
    
    private async Task<TrimSettings?> GetTrimSettingsAsync(Settings settings, List<Uri> links, List<string> localFiles, CancellationToken ct)
    {
        if (!settings.Trim) return null;

        TimeSpan? duration = null;

        // Prioritize local files for getting duration as it's faster.
        if (localFiles.Count != 0)
        {
            duration = await GetDurationFromFileAsync(localFiles.First(), ct);
        }
        // If no local files, try getting duration from the first valid link.
        else if (links.Count != 0)
        {
            duration = await GetDurationFromLinkAsync(links.First(), settings, ct);
        }

        if (duration is null || duration <= TimeSpan.Zero)
        {
            logger.Warning("Could not determine a valid video duration. Skipping trim.");
            return null;
        }

        return ShowTrimSlider(duration.Value);
    }
    
    private async Task<TimeSpan?> GetDurationFromFileAsync(string filePath, CancellationToken ct)
    {
        try
        {
            var mediaInfo = await FFmpeg.GetMediaInfo(filePath, ct);
            return mediaInfo.Duration;
        }
        catch (Exception ex)
        {
            logger.Error(ex, "Failed to get media info for {File}. Unable to determine duration for trimming.", filePath);
            return null;
        }
    }

    private async Task<TimeSpan?> GetDurationFromLinkAsync(Uri link, Settings settings, CancellationToken ct)
    {
        try
        {
            var downloadOptions = new DownloadOptions(link, settings, null);
            var runResult = await downloader.FetchMetadata(downloadOptions, ct);
            return runResult?.Data?.Duration is not null 
                ? TimeSpan.FromSeconds(runResult.Data.Duration.Value) 
                : null;
        }
        catch (Exception ex)
        {
            logger.Error(ex, "Failed to fetch metadata for {Link}. Unable to determine duration for trimming.", link);
            return null;
        }
    }

    private TrimSettings? ShowTrimSlider(TimeSpan duration)
    {
        var slider = new TrimmingSlider(duration);
        var trimResult = slider.ShowSlider();

        if (string.IsNullOrEmpty(trimResult)) return null; // User cancelled

        var parts = trimResult.Split('-');
        if (parts.Length == 2 &&
            double.TryParse(parts[0], out var start) &&
            double.TryParse(parts[1], out var end) &&
            start <= end)
        {
            return new TrimSettings(start, end - start);
        }

        logger.Warning("Invalid trim input '{TrimResult}'. Skipping trim.", trimResult);
        return null;
    }
    
    private async Task<IEnumerable<ConversionItem>> DownloadAllAsync(IEnumerable<Uri> links, Settings settings, TrimSettings? trimSettings, CancellationToken ct)
    {
        var downloadedItems = new List<ConversionItem>();
        var linkList = links.ToList();
        
        if (linkList.Count == 0) return downloadedItems;

        logger.Information("Starting download of {Count} links...", linkList.Count);
        
        foreach (var link in linkList)
        {
            ct.ThrowIfCancellationRequested();
            var downloadOptions = new DownloadOptions(link, settings, trimSettings);
            var metadata = await downloader.FetchMetadata(downloadOptions, ct);
            var result = await downloader.DownloadTask(downloadOptions, metadata, ct);

            if (result.OutPath is null)
            {
                logger.Error("Failed to download video from {Link}", link);
                continue;
            }

            AnsiConsole.MarkupLine($"Downloaded video to: [green]{result.OutPath}[/]");
            downloadedItems.Add(new ConversionItem(result.OutPath, result.fetchResult));
        }

        return downloadedItems;
    }

    private async Task ConvertAllAsync(IEnumerable<ConversionItem> items, Settings settings, TrimSettings? trimSettings, CancellationToken ct)
    {
        var itemList = items.ToList();
        if (itemList.Count == 0) return;
        
        logger.Information("Starting conversion of {Count} files...", itemList.Count);
        
        foreach (var (path, fetchResult) in itemList)
        {
            ct.ThrowIfCancellationRequested();
            try
            {
                // Note: The trimSettings here will apply to local files.
                // For downloaded files, trimming may have already been done by yt-dlp if possible
                // Applying it again with FFmpeg for local files is the intended logic
                await converter.ConvertVideo(path, fetchResult, settings, trimSettings);
                AnsiConsole.MarkupLine($"Converted video: [green]{Path.GetFileName(path)}[/]");
            }
            catch (Exception ex)
            {
                logger.Error(ex, "Failed to convert video: {Path}", path);
                AnsiConsole.MarkupLine($"[red]Failed to convert video: {Path.GetFileName(path)} - {ex.Message}[/]");
            }
        }
    }

    private void CleanupTempDirectories()
    {
        if (globals.TempDir.Count == 0) return;
        
        logger.Information("Cleaning up temporary directories...");
        globals.TempDir.ForEach(d =>
        {
            try
            {
                if (!Directory.Exists(d)) return;
                Directory.Delete(d, true);
                AnsiConsole.MarkupLine($"Deleted temp dir: [red]{d}[/]");
            }
            catch (Exception ex)
            {
                logger.Error(ex, "Failed to delete temporary directory: {Directory}", d);
            }
        });
    }

    private static async Task<bool> CheckDependenciesAsync()
    {
        var ffmpegPath = await GetCommandPathAsync("ffmpeg");
        if (string.IsNullOrWhiteSpace(ffmpegPath))
        {
            AnsiConsole.MarkupLine("[red]Error: FFmpeg not found. Please install FFmpeg and ensure it's in your system's PATH.[/]");
            return false;
        }

        var ytDlpPath = await GetCommandPathAsync("yt-dlp");
        if (!string.IsNullOrWhiteSpace(ytDlpPath)) return true;
        AnsiConsole.MarkupLine("[red]Error: yt-dlp not found. Please install yt-dlp and ensure it's in your system's PATH.[/]");
        return false;
    }

    private static async Task<string> GetCommandPathAsync(string commandName)
    {
        var processCmd = RuntimeInformation.IsOSPlatform(OSPlatform.Windows) ? "where" : "which";
        
        var processInfo = new ProcessStartInfo(processCmd, commandName)
        {
            RedirectStandardOutput = true,
            UseShellExecute = false,
            CreateNoWindow = true
        };

        using var process = Process.Start(processInfo);
        if (process is null) return string.Empty;

        var commandPath = await process.StandardOutput.ReadToEndAsync();
        await process.WaitForExitAsync();

        // 'where' can return multiple paths on newlines, we just need the first one.
        return commandPath.Split(Environment.NewLine, StringSplitOptions.RemoveEmptyEntries).FirstOrDefault()?.Trim() ?? string.Empty;
    }
}

internal record ConversionItem(string Path, RunResult<VideoData>? Metadata);