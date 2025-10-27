using dis.Features.Download.Models;
using Serilog;
using YoutubeDLSharp;
using YoutubeDLSharp.Metadata;

namespace dis.Features.Download;

public class YouTubeDownloader(YoutubeDL youtubeDl, DownloadQuery downloadQuery, ILogger logger)
    : VideoDownloaderBase(youtubeDl, downloadQuery)
{
    private const string SponsorBlockMessage = "Removing sponsored segments using SponsorBlock";

    protected override Task PreDownload(RunResult<VideoData> fetch)
    {
        var sponsorBlockEmpty = string.IsNullOrEmpty(Query.OptionSet.SponsorblockRemove);
        if (!sponsorBlockEmpty)
            logger.Information(SponsorBlockMessage);
        return Task.CompletedTask;
    }
}
