{
  description = "crook - Kubernetes node maintenance automation for Rook-Ceph clusters";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {
    flake-parts,
    self,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];

      perSystem = {
        pkgs,
        self',
        ...
      }: let
        version =
          if (self ? shortRev)
          then self.shortRev
          else "dev";
        commit =
          if (self ? rev)
          then self.rev
          else "unknown";
        # Fixed timestamp for reproducible builds (Nix requires deterministic outputs)
        buildDate = "1970-01-01T00:00:00Z";

        crook = pkgs.buildGoModule {
          pname = "crook";
          inherit version;

          src = ./.;

          # Update this hash when go.mod/go.sum changes
          # Run: nix build 2>&1 | grep "got:" to get the new hash
          vendorHash = "sha256-jH4UV9yC7E0g1PEWTDhTMzRpDFefOj9PWjidnfKO1UE=";

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
            "-X main.commit=${commit}"
            "-X main.buildDate=${buildDate}"
          ];

          subPackages = ["cmd/crook"];

          meta = with pkgs.lib; {
            description = "Kubernetes node maintenance automation for Rook-Ceph clusters";
            homepage = "https://github.com/andri/crook";
            license = licenses.mit;
            maintainers = [];
            mainProgram = "crook";
          };
        };
      in {
        packages = {
          inherit crook;
          default = crook;
        };

        apps = {
          crook = {
            type = "app";
            program = "${crook}/bin/crook";
          };
          default = self'.apps.crook;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs;
            [
              go
              git
              just
              openssl
              golangci-lint
              minikube
              kubectl
            ]
            ++ pkgs.lib.optionals stdenv.isLinux [
              docker-machine-kvm2
            ];

          shellHook = ''
            echo "crook development environment"
            echo "Go: $(go version)"
          '';
        };

        checks = {
          build = crook;
        };
      };
    };
}
