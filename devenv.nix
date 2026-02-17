{
  pkgs,
  ...
}:

{
  languages.go = {
    enable = true;
    package = pkgs.go_1_26;
  };
  packages = with pkgs; [
    golangci-lint
    github-runner
    k9s
    kind
    kubernetes-helm
    prek
    stern
    go-task
  ];
  enterTest = ''
    task test
  '';
}
