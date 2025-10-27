using System.Globalization;

namespace dis.Features.Common;

public record TrimSettings(double Start, double Duration)
{
    // Format for yt-dlp download section (uses seconds)
    public string GetDownloadSection() =>
        $"*{Start.ToString(CultureInfo.InvariantCulture)}-{(Start + Duration).ToString(CultureInfo.InvariantCulture)}";

    // Format for FFmpeg trimming (uses timestamp)
    public string GetFFmpegArgs() =>
        $"-ss {FormatTimeForFFmpeg(TimeSpan.FromSeconds(Start))} -t {FormatTimeForFFmpeg(TimeSpan.FromSeconds(Duration))}";

    // Format time for the filename (e.g., 01_500-02_000 for 1.5 s to 2 s)
    public string GetFilenamePart()
        => $"{FormatTimeForFilename(Start)}-{FormatTimeForFilename(Start + Duration)}";

    private static string FormatTimeForFFmpeg(TimeSpan time)
        => $"{(int)time.TotalHours:D2}:{time.Minutes:D2}:{time.Seconds:D2}.{time.Milliseconds:D3}";

    private static string FormatTimeForFilename(double seconds)
    {
        var timeSpan = TimeSpan.FromSeconds(seconds);
        var wholeSeconds = (int)timeSpan.TotalSeconds;
        var centiseconds = (int)(timeSpan.TotalMilliseconds % 1000) / 10;
        return $"{wholeSeconds:D2}_{centiseconds:D2}";
    }
}
