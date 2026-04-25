{ pkgs ? import
    (fetchTarball {
      name = "jpetrucciani-2026-04-22";
      url = "https://github.com/jpetrucciani/nix/archive/0e8b7fb026e5ba2f0edcd0eb411db34adf01ef24.tar.gz";
      sha256 = "0cfihvrk0zm60sfak5l9qm0szlvkg3i9wpd0k34f55969gx4q6mw";
    })
    { }
}:
let
  name = "go-api-gw";

  tools = with pkgs; {
    cli = [
      jfmt
      nixup
    ];
    go = [
      go
      go-tools
      gopls
      gcc
    ];
    scripts = pkgs.lib.attrsets.attrValues scripts;
  };

  scripts = with pkgs; { };
  paths = pkgs.lib.flatten [ (builtins.attrValues tools) ];
  env = pkgs.buildEnv {
    inherit name paths; buildInputs = paths;
  };
in
(env.overrideAttrs (_: {
  inherit name;
  NIXUP = "0.0.10";
})) // { inherit scripts; }
