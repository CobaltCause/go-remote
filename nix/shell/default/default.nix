# Keep sorted.
{
  default,
  engage,
  findutils,
  go,
  markdownlint-cli,
  mkShell,
  openssl,
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
    openssl
    reuse
    shellcheck
    shfmt
  ]
  # Keep sorted.
  ++ default.buildInputs
  ++ default.nativeBuildInputs
  ++ default.propagatedBuildInputs;
}
