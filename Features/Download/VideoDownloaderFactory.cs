using dis.Features.Download.Models;
using dis.Features.Download.Models.Interfaces;
using Serilog;
using Spectre.Console;
using YoutubeDLSharp;
using YoutubeDLSharp.Options;

namespace dis.Features.Download;

public class VideoDownloaderFactory(YoutubeDL youtubeDl, ILogger logger) : IDownloaderFactory
{
    // Constants for URL checking
    private const string YouTubeUrlPart = "youtu";
    private const string FormatSort = "vcodec:h264,ext:mp4:m4a";

    public IVideoDownloader Create(DownloadOptions o)
    {
        Dictionary<string, Func<DownloadQuery, IVideoDownloader>> customDownloadLogicDic = new()
        {
            { YouTubeUrlPart, downloadQuery => new YouTubeDownloader(youtubeDl, downloadQuery, logger) }
        };

        var optionSet = GenerateOptionSet(o);

        var entry = customDownloadLogicDic
            .FirstOrDefault(e => o.Uri.Host.Contains(e.Key));
        var query = new DownloadQuery(o.Uri, optionSet);
        return entry.Key is not null
            ? entry.Value(query)
            : new GenericDownloader(youtubeDl, query);
    }

    private static OptionSet GenerateOptionSet(DownloadOptions o)
    {
        OptionSet optionSet = new()
        {
            Output = "%(display_id)s.%(ext)s",
            FormatSort = FormatSort,
            EmbedMetadata = true
        };

        if (o.Options.SponsorBlock)
            optionSet.SponsorblockRemove = "all";

        if (o.TrimSettings is null) return optionSet;

        optionSet.ForceKeyframesAtCuts = true;
        optionSet.DownloadSections = o.TrimSettings.GetDownloadSection();

        optionSet.Output = $"%(display_id)s-{o.TrimSettings.GetFilenamePart()}.%(ext)s";

        return optionSet;
    }
}
