# Keep sorted.
{
  default,
  engage,
  findutils,
  go,
  markdownlint-cli,
  mkShell,
  reuse,
}:

mkShell {
  # Keep sorted.
  packages = [
    engage
    findutils
    go
    markdownlint-cli
    reuse
  ]
  # Keep sorted.
  ++ default.buildInputs
  ++ default.nativeBuildInputs
  ++ default.propagatedBuildInputs;
}
