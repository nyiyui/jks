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
    flake-utils.lib.eachSystem flake-utils.lib.defaultSystems (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        build-jks =
          pkgs:
          (pkgs.buildGoModule rec {
            pname = "jks";
            version = if (self ? rev) then self.rev else "dirty";
            src = ./.;
            vendorHash = "sha256-moaoaxOjcF7bV52jL/TXwoDDK3ZwIScekr4lyFrxIZo=";
            subPackages = [ "cmd/server" ];
            ldflags = [ "-X nyiyui.ca/jks/server.vcsInfo=${version}" ];
          });
      in
      rec {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            govulncheck
            nixfmt-rfc-style
            sqlite
            sqlitebrowser
            flutter
          ];
        };
        packages.default = build-jks pkgs;
        packages.jks = packages.default;
      }
    );
}
