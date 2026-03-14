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

    vendorHash = "sha256-2VWFBJToXByo5oxPSsnfsR2SsWQgrQazKjNV8T6tsZ8=";

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
