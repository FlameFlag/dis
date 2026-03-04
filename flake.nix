{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";

  outputs =
    inputs:
    let
      forAllSystems =
        f:
        inputs.nixpkgs.lib.genAttrs [
          "aarch64-linux"
          "aarch64-darwin"
          "x86_64-linux"
          "x86_64-darwin"
        ] (system: f inputs.nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {
        dis = (pkgs.callPackage ./package.nix { }).default;
        default = (pkgs.callPackage ./package.nix { }).default;
      });
    };
}
