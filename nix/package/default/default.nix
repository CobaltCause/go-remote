# Keep sorted.
{
  buildGoModule,
  protobuf,
  protoc-gen-go,
  protoc-gen-go-grpc,
  lib,
}:

buildGoModule (finalAttrs: {
  name = "go-remote";

  src =
    let
      inherit (lib.fileset) maybeMissing toSource unions;
    in
    toSource {
      root = ../../..;

      # Keep sorted.
      fileset = unions [
        ../../../bin
        ../../../cmd
        ../../../go.mod
        ../../../pkg
        ../../../proto
        (maybeMissing ../../../go.sum)
      ];
    };

  vendorHash = "sha256-Uj2wfJLb4VC6kke9w2fIw0gR7S0FTdwB5vhmkdP13RQ=";

  nativeBuildInputs = [
    protobuf
    protoc-gen-go
    protoc-gen-go-grpc
  ];

  postPatch = ''
    patchShebangs --build bin
  '';

  preBuild = ''
    ./bin/gen-proto
  '';

  meta.mainProgram = finalAttrs.name;
})
