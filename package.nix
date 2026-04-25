{
  buildGoModule,
  makeWrapper,
  yt-dlp,
  ffmpeg,
  gifski,
  lib,
  ...
}:
{
  default = buildGoModule (finalAttrs: {
    pname = "dis";
    version = "11.3.0";

    src = ./.;

    vendorHash = "sha256-hjG/qjdU0y4Tgq9+PxRIxd1KXZEqn/u6t4O7TkE8JqU=";

    ldflags = [
      "-s"
      "-w"
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
      homepage = "https://github.com/FlameFlag/dis";
      license = lib.licenses.mit;
      platforms = lib.platforms.unix;
      maintainers = [ lib.maintainers.FlameFlag ];
    };
  });
}
