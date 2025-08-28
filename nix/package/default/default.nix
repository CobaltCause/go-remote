# Keep sorted.
{
  buildGoModule,
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
        ../../../cmd
        ../../../go.mod
        (maybeMissing ../../../go.sum)
      ];
    };

  vendorHash = "sha256-GXCx7MQq1zwvQQYHvQLntHFgXbldIJY9jNyjfsxmDTQ=";

  meta.mainProgram = finalAttrs.name;
})
