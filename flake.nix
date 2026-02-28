{
  description = "trident — keyless OSINT reconnaissance CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        version = if (self ? shortRev) then self.shortRev else "dev";
      in
      {
        packages.default = pkgs.buildGo126Module {
          pname = "trident";
          inherit version;
          src = self;
          vendorHash = "sha256-OihQnAAaNlreNWCbT6ayjzIDVaYl1X7AfhKP3BmZiS8=";

          ldflags = [
            "-s" "-w"
            "-X github.com/tbckr/trident/internal/version.Version=${version}"
            "-X github.com/tbckr/trident/internal/version.Commit=${self.shortRev or "none"}"
          ];

          nativeBuildInputs = [ pkgs.installShellFiles ];
          postInstall = ''
            installShellCompletion --cmd trident \
              --bash <($out/bin/trident completion bash) \
              --zsh  <($out/bin/trident completion zsh) \
              --fish <($out/bin/trident completion fish)
          '';

          meta = with pkgs.lib; {
            description = "Keyless OSINT reconnaissance CLI";
            homepage = "https://github.com/tbckr/trident";
            license = licenses.gpl3Only;
            mainProgram = "trident";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_26
            golangci-lint
            goreleaser
          ];
        };
      }
    );
}
