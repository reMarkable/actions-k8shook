{
  pkgs,
  ...
}:

{
  languages.go.enable = true;
  packages = with pkgs; [
    github-runner
    k9s
    kind
    kubernetes-helm
    stern
  ];
}
