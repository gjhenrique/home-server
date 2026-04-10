{pkgs, ...}: {
  imports = [
    ./hardware-configuration.nix
  ];

  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;

  # Tailscale subnet router configuration
  boot.kernel.sysctl = {
    "net.ipv4.ip_forward" = 1;
    "net.ipv4.conf.all.rp_filter" = 0;
  };

  users.users.guilherme.shell = pkgs.zsh;
  users.users.guilherme.extraGroups = ["scanner"];

  services.tlp.enable = true;

   # custom modules
  services.scanbd.enable = true;
  services.scanbd.user = "guilherme";
  services.ssh-auth.enable = true;

  fileSystems."/mnt/backup_ext" = {
    device = "/dev/disk/by-uuid/fc50d3fe-3ee1-4ddb-a330-8d757bd22b00";
    fsType = "ext4";
    options = ["nofail"];
  };
}
