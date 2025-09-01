{ pkgs, lib, ... }:
{
  default = pkgs.buildDotnetModule {
    pname = "dis";
    version = "10.0.0";

    src = ./.;

    projectFile = "./dis.csproj";
    nugetDeps = ./deps.json;

    dotnet-sdk = pkgs.dotnetCorePackages.sdk_10_0;
    selfContainedBuild = true;

    executables = [ "dis" ];

    postFixup = ''
      makeWrapper ${pkgs.dotnetCorePackages.sdk_10_0}/bin/dotnet \
        "$out/bin/dis" --add-flags "$out/lib/dis/dis.dll"
      wrapProgram "$out/bin/dis" \
        --prefix PATH : ${
          lib.makeBinPath [
            pkgs.ffmpeg-full
            pkgs.yt-dlp
          ]
        } \
        --prefix LD_LIBRARY_PATH : ${lib.makeLibraryPath [ pkgs.icu ]}
    '';

    meta = {
      homepage = "https://github.com/FlameFlag/dis";
      license = lib.licenses.agpl3Plus;
      platforms = lib.platforms.unix;
      maintainers = [ lib.maintainers.FlameFlag ];
    };
  };
}
