using YoutubeDLSharp;
using YoutubeDLSharp.Metadata;

namespace dis.Features.Download.Models.Interfaces;

public interface IDownloader
{
    Task<DownloadResult> DownloadTask(DownloadOptions options, RunResult<VideoData>? fetchResult, CancellationToken ct);
    Task<RunResult<VideoData>?> FetchMetadata(DownloadOptions options, CancellationToken ct);
}
