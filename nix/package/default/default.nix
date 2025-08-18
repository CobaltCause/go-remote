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

  # Set this to avoid having to update a hash all the time. Unfortunately this
  # means dependencies need to be vendored.
  vendorHash = null;

  meta.mainProgram = finalAttrs.name;
})
