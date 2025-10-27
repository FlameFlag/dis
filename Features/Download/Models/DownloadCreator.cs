using dis.Features.Common;
using dis.Features.Download.Models.Interfaces;
using Spectre.Console;
using YoutubeDLSharp;
using YoutubeDLSharp.Metadata;

namespace dis.Features.Download.Models;

public sealed class DownloadCreator(Globals globals, IDownloaderFactory factory) : IDownloader
{
    public async Task<DownloadResult> DownloadTask(DownloadOptions options, RunResult<VideoData>? fetchResult,
        CancellationToken ct)
    {
        PrepareTempDirectory();

        var videoDownloader = factory.Create(options);

        var dlResult = await videoDownloader.Download(fetchResult);
        return dlResult.OutPath is null
            ? new DownloadResult(null, fetchResult)
            : dlResult;
    }

    public async Task<RunResult<VideoData>?> FetchMetadata(DownloadOptions options, CancellationToken ct)
    {
        var result = await globals.YoutubeDl.RunVideoDataFetch(options.Uri.ToString(), ct);
        if (!result.Success)
        {
            AnsiConsole.MarkupLine("[red]Failed to fetch video data[/]");
            throw new Exception();
        }
        if (result.Data.Duration.HasValue) return result;

        AnsiConsole.MarkupLine("[red]Video has no duration[/]");
        throw new Exception();
    }

    /// <summary>
    /// Prepare a TEMP folder for downloading a video
    /// </summary>
    private void PrepareTempDirectory()
    {
        var temp = Path.GetTempPath();
        var folderName = Guid.NewGuid().ToString()[..6];
        var tempPath = Path.Combine(temp, folderName);

        Directory.CreateDirectory(tempPath);

        globals.YoutubeDl.OutputFolder = tempPath;
        globals.TempDir.Add(tempPath);
    }
}
