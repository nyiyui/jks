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
          version = if (self ? rev) then self.rev else "dirty";
          src = ./.;
          vendorHash = "sha256-7LfWETUR3A6SKuLoT8vAsem6zI5hl9OvE0HqxZtCXNQ=";
          subPackages = [ "cmd/server" ];
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
