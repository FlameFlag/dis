using System.Globalization;
using dis.Features.Common;
using dis.Features.Conversion.Models;
using Serilog;
using Spectre.Console;
using Xabe.FFmpeg;
using YoutubeDLSharp;
using YoutubeDLSharp.Metadata;

namespace dis.Features.Conversion;

public sealed class Converter(
    PathHandler pathHandler,
    ValidResolutions validResolutions,
    ProcessHandler processHandler,
    ILogger logger)
{
    /// <summary>
    /// Converts a video file to a specified format using FFmpeg.
    /// </summary>
    /// <param name="file">The path to the input video file.</param>
    /// <param name="fetchResult">The fetched video data.</param>
    /// <param name="s">The options to use for the conversion.</param>
    /// <param name="trimSettings">The optional trim settings for the conversion.</param>
    /// <returns>A task that represents the asynchronous conversion operation.</returns>
    public async Task ConvertVideo(string file, RunResult<VideoData>? fetchResult, Settings s, TrimSettings? trimSettings)
    {
        while (true)
        {
            Console.CancelKeyPress += HandleCancellation;

            var cmpPath = pathHandler.GetCompressPath(file, s.VideoCodec);
            var outP = pathHandler.ConstructFilePath(s, cmpPath);

            var mediaInfo = await FFmpeg.GetMediaInfo(file);
            var streams = mediaInfo.Streams.ToList();

            var conversion = processHandler.ConfigureConversion(s, streams, outP, trimSettings);
            if (conversion is null)
            {
                logger.Error("Could not configure conversion");
                return;
            }

            try
            {
                await AnsiConsole.Status()
                    .StartAsync("Starting conversion...", async ctx =>
                    {
                        ctx.Spinner(Spinner.Known.Arrow);

                        conversion.OnProgress += (_, args) =>
                        {
                            var percent = (int)Math.Round(args.Duration.TotalSeconds / args.TotalLength.TotalSeconds * 100);
                            if (percent is 0) return;

                            ctx.Status($"[green]Conversion progress: {percent}%[/]");
                            ctx.Refresh();
                        };
                        await conversion.Start();
                    });
                if (fetchResult?.Data.UploadDate != null)
                {
                    File.SetCreationTime(outP, fetchResult.Data.UploadDate.Value);
                    File.SetLastWriteTime(outP, fetchResult.Data.UploadDate.Value);
                    File.SetLastAccessTime(outP, fetchResult.Data.UploadDate.Value);
                }
            }
            catch (Exception)
            {
                logger.Error("Conversion failed");
                logger.Error("FFmpeg args: {Conversion}", $"ffmpeg {conversion.Build().Trim()}");
                return;
            }

            logger.Information("Converted video saved at: {Path}", outP);

            var (originalSize, compressedSize) = ResultsTable(file, outP);
            if (compressedSize > originalSize)
            {
                var videoStream = streams.OfType<IVideoStream>().FirstOrDefault();
                if (videoStream is not null)
                    if (ShouldRetry(s, outP, streams)) continue;
            }

            break;
        }
    }

    private static (double, double) ResultsTable(string file, string outP)
    {
        var originalSize = (double)new FileInfo(file).Length;
        var compressedSize = (double)new FileInfo(outP).Length;

        var saved = originalSize - compressedSize;
        var savedPercentRounded = GetSavedPercentRounded(saved, originalSize);

        var originalSizeString = GetSizeString(originalSize, "Original size: [{0}]{1}[/]");
        var compressedSizeString = GetSizeString(compressedSize,
            "Compressed size: [{0}]{1}[/]",
            compressedSize > originalSize
                ? "red"
                : "green");

        var savedString = GetSavedString(saved, savedPercentRounded);

        Table table = new();
        table.AddColumn("Original");
        table.AddColumn("Compressed");
        table.AddColumn(saved < 0 ? "Increased" : "Saved");
        table.AddRow(originalSizeString, compressedSizeString, savedString);

        AnsiConsole.Write(table);
        return (originalSize, compressedSize);
    }

    private static string GetSavedPercentRounded(double saved, double originalSize)
    {
        var savedPercent = saved / originalSize * 100;
        return (savedPercent > 0
                   ? "-"
                   : "+") +
               Math.Round(Math.Abs(savedPercent),
                       2)
                   .ToString(CultureInfo.InvariantCulture);
    }

    private static string GetSizeString(double size, string format, string color = "red")
    {
        var sizeMiB = size / 1024.0 / 1024.0;
        var sizeKiB = size / 1024.0;
        var sizeString = sizeMiB < 1
            ? string.Format(format, color, $"{Math.Round(sizeKiB, 2):F2} KiB")
            : string.Format(format, color, $"{Math.Round(sizeMiB, 2):F2} MiB");
        return sizeString;
    }

    private static string GetSavedString(double saved, string savedPercentRounded)
    {
        var savedMiB = Math.Abs(saved) / 1024.0 / 1024.0;
        var savedKiB = Math.Abs(saved) / 1024.0;
        var savedColor = saved < 0 ? "red" : "green";
        var savedChange = saved < 0 ? "Increased" : "Saved";
        var savedSymbol = saved < 0 ? "+" : "-";
        var savedSizeString = savedMiB < 1
            ? $"{savedSymbol}{Math.Round(savedKiB, 2):F2} KiB"
            : $"{savedSymbol}{Math.Round(savedMiB, 2):F2} MiB";
        return $"{savedChange}: [{savedColor}]{savedSizeString} ({savedPercentRounded}%)[/]";
    }

    private bool ShouldRetry(Settings s, string outP, IEnumerable<IStream> streams)
    {
        logger.Warning("The resulting file is larger than the original.");

        var deleteAndRetry = AskForRetry();
        if (!deleteAndRetry) return false;
        if (File.Exists(outP))
        {
            File.Delete(outP);
            logger.Information("Deleted the oversized converted video.");
        }

        var resolutionChanged = AskForResolutionChange(streams, s);
        var crfChanged = AskForCrfChange(s);

        return resolutionChanged || crfChanged;
    }

    private static bool AskForRetry()
    {
        var deleteAndRetry = AnsiConsole.Prompt(
            new SelectionPrompt<string>()
                .Title("Do you want to delete the converted video and try again with a better setting?")
                .AddChoices("Yes", "No"));

        return deleteAndRetry is "Yes";
    }

    private bool AskForResolutionChange(IEnumerable<IStream> streams, Settings s)
    {
        var resolutionChanged = false;
        var changeResolution = AnsiConsole.Prompt(
            new SelectionPrompt<string>()
                .Title("Would you like to change the resolution?")
                .AddChoices("Yes", "No"));

        if (changeResolution is "Yes") resolutionChanged = ChangeResolution(streams, s);
        return resolutionChanged;
    }

    private bool ChangeResolution(IEnumerable<IStream> streams, Settings s)
    {
        var videoStream = streams.OfType<IVideoStream>().First();
        var width = videoStream.Width;
        var height = videoStream.Height;

        var maxDimension = height > width ? height : Math.Max(width, height);
        var currentResolution = GetClosestResolution(maxDimension);

        var resolutionList = validResolutions.Resolutions.ToList();
        var currentResolutionIndex = resolutionList.IndexOf(currentResolution);

        if (currentResolutionIndex <= 0) return false;

        var lowerResolutions = resolutionList.GetRange(0, currentResolutionIndex);
        var chosenResolution = AnsiConsole.Prompt(
            new SelectionPrompt<string>()
                .Title("Please select a lower resolution for the conversion:")
                .AddChoices(lowerResolutions.Select(res => res + "p")));

        s.Resolution = chosenResolution;
        return true;
    }

    private int GetClosestResolution(int dimension)
    {
        var closestResolutionIndex = validResolutions.Resolutions.BinarySearch(dimension);

        // If dimension is not found, BinarySearch returns a negative number that is the bitwise complement 
        // of the index of the next element that is larger than dimension or, if there is no larger element, 
        // the bitwise complement of Count. So we have to handle this case and revert it to the correct value.
        if (closestResolutionIndex < 0) closestResolutionIndex = ~closestResolutionIndex - 1;

        // if there's no resolution smaller than dimension, choose the smallest one
        if (closestResolutionIndex < 0) closestResolutionIndex = 0;

        return validResolutions.Resolutions[closestResolutionIndex];
    }

    private static bool AskForCrfChange(Settings s)
    {
        var crfChanged = false;
        var crfAnswer = AnsiConsole.Prompt(
            new SelectionPrompt<string>()
            .Title("Would you like to enter a new crf value?")
            .AddChoices("Yes", "No"));

        if (crfAnswer is "Yes") crfChanged = ChangeCrfValue(s);

        return crfChanged;
    }

    private static bool ChangeCrfValue(Settings s)
    {
        bool crfChanged;

        while (true)
        {
            var crfAnswer = AnsiConsole.Ask<string>("Please enter new value:");
            var crfValue = int.Parse(new string(crfAnswer.Where(char.IsDigit).ToArray()));

            if (crfValue <= s.Crf)
            {
                AnsiConsole.WriteLine("Please enter a value higher than the current CRF.");
                continue;
            }

            s.Crf = crfValue;
            crfChanged = true;
            break;
        }

        return crfChanged;
    }

    private static void HandleCancellation(object? sender, ConsoleCancelEventArgs e)
    {
        if (e.SpecialKey is not ConsoleSpecialKey.ControlC)
            return;

        AnsiConsole.WriteLine("Canceled");
    }
}
