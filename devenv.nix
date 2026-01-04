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
  ];

  languages.go = {
    enable = true;
  };
}
