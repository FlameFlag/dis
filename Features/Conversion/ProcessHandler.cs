using dis.Features.Common;
using Serilog;
using Xabe.FFmpeg;

namespace dis.Features.Conversion;

public sealed class ProcessHandler(ILogger logger, CodecParser codecParser, StreamConfigurator configurator)
{
    private const string NoStreamError = "There is no video or audio stream in the file";
    private const string FastStartParam = "-movflags +faststart";

    public IConversion? ConfigureConversion(Settings o, IList<IStream> streams, string outP, TrimSettings? trimSettings)
    {
        var videoStream = streams.OfType<IVideoStream>().FirstOrDefault();
        var audioStream = streams.OfType<IAudioStream>().FirstOrDefault();

        if (videoStream is null && audioStream is null)
        {
            logger.Error(NoStreamError);
            return default;
        }

        var parameters = new List<string> { $"-crf {o.Crf}" };

        // Add trim parameters first if provided
        if (trimSettings is not null)
            parameters.Insert(0, trimSettings.GetFFmpegArgs());

        // Add faststart for web playback formats
        var isWebPlayBackFormat = o.VideoCodec is "libx264" or "h264" ||
                                 outP.EndsWith(".mp4", StringComparison.OrdinalIgnoreCase) ||
                                 outP.EndsWith(".mov", StringComparison.OrdinalIgnoreCase);
        if (isWebPlayBackFormat)
        {
            parameters.Add(FastStartParam);
        }

        var conversion = FFmpeg.Conversions.New()
            .AddParameter(string.Join(" ", parameters))
            .SetPixelFormat(PixelFormat.yuv420p)
            .SetPreset(ConversionPreset.VerySlow);

        ConfigureVideoStream(conversion, videoStream, o);
        ConfigureAudioStream(conversion, audioStream, o);

        conversion.SetOutput(outP);
        return conversion;
    }

    private void ConfigureVideoStream(IConversion conversion, IVideoStream? videoStream, Settings o)
    {
        if (videoStream is null) return;

        var videoCodec = codecParser.GetCodec(o.VideoCodec);
        conversion.AddStream(videoStream);

        switch (videoCodec)
        {
            case VideoCodec.vp9:
                configurator.SetVp9Args(conversion);
                break;

            case VideoCodec.av1:
                configurator.SetCpuForAv1(conversion, videoStream.Framerate);
                conversion.SetPixelFormat(PixelFormat.yuv420p10le);
                break;

            default:
                conversion.UseMultiThread(o.MultiThread);
                break;
        }

        if (!string.IsNullOrEmpty(o.Resolution))
            configurator.SetResolution(videoStream, o.Resolution);
    }

    private static void ConfigureAudioStream(IConversion conversion, IAudioStream? audioStream, Settings o)
    {
        if (audioStream is null) return;

        if (o.AudioBitrate.HasValue)
            conversion.SetAudioBitrate((long)(o.AudioBitrate * 1000));

        var audioCodec = o.VideoCodec is "vp8" or "vp9" or "av1" ? AudioCodec.libopus : AudioCodec.aac;
        audioStream.SetCodec(audioCodec);
        conversion.AddStream(audioStream);
    }
}
