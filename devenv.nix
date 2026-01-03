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
  ];

  languages.go = {
    enable = true;
  };
}
