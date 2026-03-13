{
  buildGoModule,
  makeWrapper,
  yt-dlp,
  ffmpeg-full,
  lib,
  ...
}:
{
  default = buildGoModule (finalAttrs: {
    pname = "dis";
    version = "11.0.0";

    src = ./.;

    vendorHash = "sha256-UzDkS/dOCvLf8D/tUht70wTvhC15NigsS8e8J7wkNvI=";

    ldflags = [
      "-s"
      "-w"
    ];

    nativeBuildInputs = [ makeWrapper ];

    postInstall = ''
      wrapProgram "$out/bin/dis" \
        --prefix PATH : ${
          lib.makeBinPath [
            ffmpeg-full
            yt-dlp
          ]
        }
    '';

    meta = {
      homepage = "https://github.com/FlameFlag/dis";
      license = lib.licenses.mit;
      platforms = lib.platforms.unix;
      maintainers = [ lib.maintainers.FlameFlag ];
    };
  });
}
