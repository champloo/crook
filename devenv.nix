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
    docker-machine-kvm2
  ];

  languages.go = {
    enable = true;
  };
}
