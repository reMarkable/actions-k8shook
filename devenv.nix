{
  pkgs,
  ...
}:

{
  languages.go.enable = true;
  packages = with pkgs; [
    kind
    github-runner
    kubernetes-helm
  ];
}
