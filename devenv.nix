{
  pkgs,
  inputs,
  ...
}:
{
  packages = with pkgs; [
    git
    just
    openssl
    golangci-lint
    minikube
    kubectl
  ];

  languages.go = {
    enable = true;
  };
}
