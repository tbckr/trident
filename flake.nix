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
        commit = self.shortRev or "none";
        lmd = self.lastModifiedDate or "19700101000000";
        date = "${builtins.substring 0 4 lmd}-${builtins.substring 4 2 lmd}-${builtins.substring 6 2 lmd}T${builtins.substring 8 2 lmd}:${builtins.substring 10 2 lmd}:${builtins.substring 12 2 lmd}Z";
      in
      {
        packages.default = pkgs.buildGo126Module {
          pname = "trident";
          inherit version;
          src = self;
          vendorHash = "sha256-OihQnAAaNlreNWCbT6ayjzIDVaYl1X7AfhKP3BmZiS8=";

          ldflags = [
            "-s" "-w"
            "-X github.com/tbckr/trident/internal/version.Commit=${commit}"
            "-X github.com/tbckr/trident/internal/version.Date=${date}"
          ];

          nativeBuildInputs = [ pkgs.installShellFiles ];

          postBuild = ''
            go build -o $TMPDIR/docgen ./cmd/docgen
            $TMPDIR/docgen --version=${version} --output-dir=$TMPDIR/docs
          '';

          postInstall = ''
            installManPage $TMPDIR/docs/man/*.1

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
