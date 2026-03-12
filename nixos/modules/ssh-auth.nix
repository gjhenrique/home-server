# Copied from https://github.com/Cody-W-Tucker/nix-config/blob/bc93a321127d40a5d857736c92fcc148c2859e61/modules/server/paperless-scanning.nix

{ config, lib, pkgs, ... }:

with lib;

{
  options = {
    services.ssh-auth.enable = mkOption {
      type = types.bool;
      default = false;
      description = ''
        Add SSH keys authentication
      '';
    };
  };

  config = mkIf config.services.ssh-auth.enable {
    users.users.guilherme.openssh.authorizedKeys.keys = [
      "sk-ssh-ed25519@openssh.com AAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29tAAAAIJukjvvy7oit8HPiS3alfmjoxLflH6cg9ZZXg2i0PApfAAAABHNzaDo= guilherme@dell"
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIM/HLFVWHMqNfNcN7cQrnWxgP7r8G18JTkHcaWnnavpR SSH Key"
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIALVCokcHsh/mRxP92nbldzFZFWPy+OggUsZaa5uvtGt guilherme@dell"
    ];

    services.openssh = {
      enable = true;
      ports = [ 2200 ];
      settings = {
        PasswordAuthentication = false;
        AllowUsers = [ "guilherme" ];
        UseDns = true;
        X11Forwarding = false;
        PermitRootLogin = "prohibit-password";
      };
    };
  };
}
