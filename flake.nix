{
  # Based upon templates#go-hello
  description = "Concur";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      # System types to support (not tested).
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          concur =
            pkgs.buildGoModule {
              pname = "concur";
              src = ./.;
              version = builtins.readFile ./.version;

              # Must be updated when dependencies are updated
              vendorHash = "sha256-SESMSCNoiKu0aUyZhatMWyGnd9Q+qlnTOG274m3ydCI="; 

              doCheck = false;

              meta = with pkgs.lib; {
                description = "A replacement for the parts of GNU Parallel that I like";
                homepage = "https://github.com/ewosborne/concur";
                license = licenses.gpl3;
                maintainers = with maintainers; [ ]; # TODO
              };
            };
      });

      # The default package for 'nix build'. This makes sense if the
      # flake provides only one package or there is a clear "main"
      # package.
      defaultPackage = forAllSystems (system: self.packages.${system}.concur);
    };
}
