{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      ...
    }@attrs:
    let
      pkgs = import nixpkgs { config.allowUnfree = true; };
    in
    flake-utils.lib.eachSystem flake-utils.lib.defaultSystems (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        build-jks = pkgs: (pkgs.buildGoModule rec {
          pname = "jks";
          version = "0.0.0";
          src = ./.;
          vendorHash = "sha256-dwSvxFceSNvoGqbSjAXmIFElVMhgK4od0V2ij/GYje0=";
        });
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            nixfmt-rfc-style
            sqlite
            sqlitebrowser
            flutter
          ];
          nativeBuildInputs = with pkgs; [
            pkg-config
            libGL
            xorg.libX11.dev
            xorg.libXcursor
            xorg.libXi
            xorg.libXinerama
            xorg.libXrandr
            xorg.libXxf86vm
          ];
        };
        packages.jks = build-jks pkgs;
      }
    );
}
