{ sprinkles ? null }:

let
  source = import ./lon.nix;

  # Keep sorted.
  input = source: {
    nixpkgs = import source.nixpkgs {
      config.allowAliases = false;
    };
    sprinkles = if sprinkles == null
      then import source.sprinkles
      else sprinkles;
  };
in

(input source).sprinkles.new {
  inherit input source;

  output = self:
    let
      inherit (self.input) nixpkgs;
      inherit (self.input.nixpkgs.lib.customisation) makeScope;
    in
    {
      package = makeScope nixpkgs.newScope (scope: {
        default = scope.callPackage ./package/default {};
      });

      shell = makeScope self.output.package.newScope (scope: {
        default = scope.callPackage ./shell/default {
          inherit (self.output.package) default;
        };
      });
    };
}
