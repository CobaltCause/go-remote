# Keep sorted.
{
  default,
  engage,
  findutils,
  go,
  markdownlint-cli,
  mkShell,
  reuse,
  shellcheck,
  shfmt,
}:

mkShell {
  # Keep sorted.
  packages = [
    engage
    findutils
    go
    markdownlint-cli
    reuse
    shellcheck
    shfmt
  ]
  # Keep sorted.
  ++ default.buildInputs
  ++ default.nativeBuildInputs
  ++ default.propagatedBuildInputs;
}
