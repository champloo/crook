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
  ];

  languages.go = {
    enable = true;
  };
}
