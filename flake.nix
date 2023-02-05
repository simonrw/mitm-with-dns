{
  description = "Flake utils demo";

  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            delve
            lldb
            llvm
          ];

          hardeningDisable = [ "fortify" ];
          LLVM_SYMBOLIZER_PATH = "${pkgs.llvm}/bin/llvm-symbolizer";
        };
      }
    );
}
