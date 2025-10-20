{
  pkgs,
  ...
}:

{
  languages.go.enable = true;
  packages = with pkgs; [
    kind
    k9s
    github-runner
    kubernetes-helm
  ];
}
