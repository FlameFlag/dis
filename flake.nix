{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/master";

  outputs =
    inputs:
    let
      version =
        let
          date = inputs.self.lastModifiedDate or "19700101000000";
          formattedDate = builtins.concatStringsSep "-" [
            (builtins.substring 0 4 date)
            (builtins.substring 4 2 date)
            (builtins.substring 6 2 date)
          ];
          revision = inputs.self.shortRev or "dirty";
        in
        "0-unstable-${formattedDate}-${revision}";
      packageFor = pkgs: (pkgs.callPackage ./package.nix { inherit version; }).default;
      forAllSystems =
        f:
        inputs.nixpkgs.lib.genAttrs [
          "aarch64-linux"
          "aarch64-darwin"
          "x86_64-linux"
        ] (system: f inputs.nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {
        dis = packageFor pkgs;
        default = packageFor pkgs;
      });
    };
}
