{
  description = "Flake utils demo";

  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in
      {
        devShells.default = pkgs.mkShellNoCC {
          buildInputs = with pkgs; [
            go
            gopls
            delve
            lldb
            llvm
            mkcert
            gnumake
          ];

          CAROOT = "./ca";
        };
      }
    );
}
