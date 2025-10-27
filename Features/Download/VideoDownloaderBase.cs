using dis.Features.Download.Models;
using dis.Features.Download.Models.Interfaces;
using Serilog;
using Spectre.Console;
using YoutubeDLSharp;
using YoutubeDLSharp.Metadata;
using dis.Features.Common;

namespace dis.Features.Download;

public abstract class VideoDownloaderBase(YoutubeDL youtubeDl, DownloadQuery query) : IVideoDownloader
{
    protected readonly YoutubeDL YoutubeDl = youtubeDl;
    protected readonly DownloadQuery Query = query;
    private readonly ILogger _logger = Log.Logger.ForContext<VideoDownloaderBase>();

    private const string LiveStreamError = "Live streams are not supported";
    private const string DownloadError = "Download failed";
    private const string FetchError = "Failed to fetch url";

    public async Task<DownloadResult> Download(RunResult<VideoData>? fetchResult)
    {
        var fetch = fetchResult ?? await FetchVideoData();
        if (fetch is null || !fetch.Success)
        {
            _logger.Error(FetchError);
            return new DownloadResult(null, fetchResult);
        }

        if (fetch.Data.IsLive is true)
        {
            _logger.Error(LiveStreamError);
            return new DownloadResult(null, fetchResult);
        }

        // Pre-download custom logic
        await PreDownload(fetch);

        // The downloading part can be overridden in child classes
        var dlResult = await DownloadVideo();
        if (dlResult is null)
            return new DownloadResult(null, fetchResult);
        if (dlResult.Success is false)
        {
            _logger.Error(DownloadError);
            return new DownloadResult(null, fetchResult);
        }

        // Post-download custom logic
        var postDownload = await PostDownload(fetch);
        var postNull = string.IsNullOrEmpty(postDownload);

        var path = postNull
            ? Directory.GetFiles(YoutubeDl.OutputFolder).FirstOrDefault()
            : postDownload;
        return new DownloadResult(path, fetchResult);
    }

    private async Task<RunResult<VideoData>?> FetchVideoData()
    {
        RunResult<VideoData>? fetch = null;
        await AnsiConsole.Status().StartAsync("Fetching data...", async ctx =>
        {
            ctx.Spinner(Spinner.Known.Arrow);
            fetch = await YoutubeDl.RunVideoDataFetch(Query.Uri.ToString());
            ctx.Refresh();
        });
        return fetch;
    }

    protected virtual Task PreDownload(RunResult<VideoData> fetch)
        => Task.CompletedTask;

    protected virtual Task<string> PostDownload(RunResult<VideoData> fetch)
        => Task.FromResult(string.Empty);

    private async Task<RunResult<string?>?> DownloadVideo()
    {
        RunResult<string?>? download = null;
        await AnsiConsole.Status().StartAsync("Downloading...", async ctx =>
        {
            ctx.Spinner(Spinner.Known.Arrow);

            Query.OptionSet.Output = Path.Combine(YoutubeDl.OutputFolder, Query.OptionSet.Output);

            download = await YoutubeDl.RunVideoDownload(Query.Uri.ToString(),
                overrideOptions: Query.OptionSet,
                progress: new Progress<DownloadProgress>(p =>
                {
                    var progress = (int)Math.Round(p.Progress * 100);
                    if (progress is 0 or 1)
                        return;

                    ctx.Status($"[green]Download Progress: {progress}%[/]");
                    ctx.Refresh();
                }));
        });
        return download;
    }
}
