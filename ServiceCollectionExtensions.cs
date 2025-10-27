using dis.Features.Common;
using dis.Features.Conversion;
using dis.Features.Conversion.Models;
using dis.Features.Download;
using dis.Features.Download.Models;
using dis.Features.Download.Models.Interfaces;
using Microsoft.AspNetCore.StaticFiles;
using Microsoft.Extensions.DependencyInjection;
using Serilog;
using Serilog.Core;

namespace dis;

public static class ServiceCollectionExtensions
{
    public static void AddMyServices(this IServiceCollection services)
    {
        services.AddSingleton<Globals>();
        services.AddSingleton<ValidResolutions>();
        services.AddSingleton<VideoCodecs>();
        services.AddSingleton<FileExtensionContentTypeProvider>();
        services.AddSingleton<RootCommand>();
        services.AddSingleton<IDownloaderFactory, VideoDownloaderFactory>();

        services.AddTransient<CodecParser>();
        services.AddTransient<StreamConfigurator>();
        services.AddTransient<ProcessHandler>();
        services.AddTransient<PathHandler>();
        services.AddTransient<Converter>();
        services.AddTransient<LoggingLevelSwitch>();

        services.AddTransient<IDownloader, DownloadCreator>();
        services.AddSingleton<IDownloaderFactory>(sp =>
        {
            var globals = sp.GetRequiredService<Globals>();
            var logger = sp.GetRequiredService<ILogger>();
            return new VideoDownloaderFactory(globals.YoutubeDl, logger.ForContext<VideoDownloaderFactory>());
        });
    }
}
