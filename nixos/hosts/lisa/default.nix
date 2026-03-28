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

  # Encrypted external backup disk
  environment.etc."luks-keys/backup_ext" = {
    source = ./backup_ext.key;
    mode = "0400";
  };

  environment.etc.crypttab.text = ''
    backup_ext UUID=5dabbf9a-135f-4efa-8293-4d2680a2dddc /etc/luks-keys/backup_ext nofail
  '';

  services.udev.extraRules = ''
    ACTION=="add|change", ENV{ID_FS_UUID}=="5dabbf9a-135f-4efa-8293-4d2680a2dddc", ENV{ID_FS_TYPE}=="crypto_LUKS", ENV{SYSTEMD_WANTS}+="systemd-cryptsetup@backup_ext.service", TAG+="systemd"
    ACTION=="add|change", ENV{DM_NAME}=="backup_ext", ENV{SYSTEMD_WANTS}+="mnt-backup_ext.mount", TAG+="systemd"
  '';

  fileSystems."/mnt/backup_ext" = {
    device = "/dev/mapper/backup_ext";
    fsType = "ext4";
    options = [ "nofail" ];
  };
}
