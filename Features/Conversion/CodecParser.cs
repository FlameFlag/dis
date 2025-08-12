using dis.Features.Conversion.Models;
using Xabe.FFmpeg;

namespace dis.Features.Conversion;

public sealed class CodecParser(VideoCodecs videoCodecs)
{
    /// <summary>
    /// Retrieves the type of video codec from a given string
    /// </summary>
    /// <param name="inputCodec">The string that represents the video codec.</param>
    /// <returns>The type of VideoCodec that matches the given string, or VideoCodec.libx264 if no match is found.</returns>
    public VideoCodec GetCodec(string inputCodec) => ParseCodec(inputCodec);

    /// <summary>
    /// Checks the given string to find out the type of video codec.
    /// If no match is found or if the string is empty, it returns the default video codec 'libx264'.
    /// </summary>
    /// <param name="inputCodec">The string that represents the video codec.</param>
    /// <returns>The type of VideoCodec that matches the given string, or VideoCodec.libx264 if no match is found.</returns>
    private VideoCodec ParseCodec(string inputCodec)
    {
        // If the input string is null, return the default video codec
        if (string.IsNullOrWhiteSpace(inputCodec))
            return VideoCodec.libx264;

        foreach (var (key, value) in videoCodecs.Codecs)
        {
            // If the key does not contain the input string, continue to the next iteration
            if (key.Contains(inputCodec) is false)
                continue;

            return value;
        }

        // If no match is found in the dictionary, return the default video codec
        return VideoCodec.libx264;
    }
}
