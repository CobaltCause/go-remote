# Keep sorted.
{
  default,
  engage,
  findutils,
  go,
  markdownlint-cli,
  mkShell,
  protobuf,
  protoc-gen-go,
  reuse,
}:

mkShell {
  # Keep sorted.
  packages = [
    engage
    findutils
    go
    markdownlint-cli
    protobuf
    protoc-gen-go
    reuse
  ]
  # Keep sorted.
  ++ default.buildInputs
  ++ default.nativeBuildInputs
  ++ default.propagatedBuildInputs;
}
