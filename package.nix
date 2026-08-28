{
  buildGo127Module,
  makeWrapper,
  yt-dlp,
  ffmpeg,
  gifski,
  lib,
  version,
  ...
}:
{
  default = buildGo127Module (finalAttrs: {
    pname = "dis";
    inherit version;

    src = ./.;

    vendorHash = "sha256-z19l7/L7/1w8PvLPfIBjv98GlFqPuPm6x8qYQQ1yHCA=";

    ldflags = [
      "-s"
      "-w"
      "-X main.version=${finalAttrs.version}"
    ];

    nativeBuildInputs = [ makeWrapper ];

    postInstall = ''
      wrapProgram "$out/bin/dis" \
        --prefix PATH : ${
          lib.makeBinPath [
            ffmpeg
            yt-dlp
            gifski
          ]
        }
    '';

    meta = {
      description = "CLI and TUI for downloading, trimming, and compressing videos";
      homepage = "https://github.com/4evy/dis";
      license = lib.licenses.mit;
      mainProgram = "dis";
      platforms = lib.platforms.unix;
      maintainers = [ lib.maintainers._4evy ];
    };
  });
}
