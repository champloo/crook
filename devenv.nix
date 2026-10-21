{
  pkgs,
  inputs,
  ...
}:
let
  llm-agents = inputs.llm-agents.packages.${pkgs.stdenv.system};
in
{
  packages = with pkgs; [
    git
    just
    openssl
    golangci-lint
    minikube
    kubectl
    llm-agents.beads-rust
    llm-agents.beads-viewer
    # br doctor hands off WAL recovery to the stock sqlite3 CLI
    sqlite
  ];

  languages.go = {
    enable = true;
  };
}
