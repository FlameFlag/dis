using dis.Features.Common;
using Xabe.FFmpeg;

namespace dis.Features.Conversion;

public sealed class PathHandler(CodecParser codecParser)
{
    private readonly Dictionary<string, VideoCodec[]> _videoExtMap = new()
    {
        { "mp4", [VideoCodec.libx264, VideoCodec.hevc] },
        { "webm", [VideoCodec.vp8, VideoCodec.vp9, VideoCodec.av1] }
    };

    /// <summary>
    /// This method returns the path of the compressed file with the appropriate extension based on the codec.
    /// </summary>
    /// <param name="file">The original file path.</param>
    /// <param name="codec">The codec used for compression, can be null.</param>
    /// <returns>The new file path with the appropriate extension for the compressed file.</returns>
    public string GetCompressPath(string file, string codec)
    {
        var videoCodec = codecParser.GetCodec(codec);

        var matchingExtensions = _videoExtMap
            .Where(kvp => kvp.Value.Contains(videoCodec))
            .ToList();

        string? extension = null;
        if (matchingExtensions.Count == 0)
            return Path.ChangeExtension(file, extension);

        // If the file already has the correct extension, return the file path
        extension = matchingExtensions.First().Key;

        return Path.ChangeExtension(file, extension);
    }

    /// <summary>
    /// Constructs the file path for the output file based on the provided options and the path of the compressed file.
    /// </summary>
    /// <param name="settings">The parsed options for file processing.</param>
    /// <param name="cmpPath">The path of the compressed file.</param>
    /// <returns>The constructed file path for the output file.</returns>
    /// <exception cref="Exception">Thrown when the original file name cannot be retrieved from the compressed file path.</exception>
    public string ConstructFilePath(Settings settings, string cmpPath)
    {
        var id = Guid.NewGuid().ToString()[..4];

        var orgName = Path.GetFileName(cmpPath);

        if (orgName is null)
            throw new FileNotFoundException("Could not get the original file name");

        var iniOutPath = Path.Combine(settings.Output!, orgName);

        var orgExt = Path.GetExtension(cmpPath);

        var outName = File.Exists(iniOutPath)
            // If a file exists, append the id to the original file name
            ? $"{Path.GetFileNameWithoutExtension(orgName)}-{id}{orgExt}"
            // If no file exists, use the original file name
            : orgName;

        // If random file names are enabled, use the id as the file name
        if (settings.RandomFileName)
            outName = $"{id}{orgExt}";

        return Path.Combine(settings.Output!, outName);
    }
}
