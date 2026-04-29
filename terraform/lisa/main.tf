terraform {
  required_version = ">= 1.6"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5"
    }
  }
}

locals {
  secrets = yamldecode(file("${path.module}/secrets.yaml"))

  public_subdomains = [
    "@",
    "actual",
    "auth",
    "backup",
    "dns",
    "docs",
    "dvd",
    "emby",
    "ha",
    "pdf",
    "photos",
    "scan",
  ]

  subdomains = concat(local.public_subdomains, local.secrets.subdomains)

  records = [
    for s in local.subdomains : {
      name    = s
      type    = "A"
      content = local.secrets.ip
    }
  ]

  records_by_key = {
    for r in local.records :
    "${r.type}-${r.name}" => r
  }
}

provider "cloudflare" {
  api_token = local.secrets.cloudflare_api_token
}

data "cloudflare_zone" "this" {
  filter = {
    name = local.secrets.domain
  }
}

resource "cloudflare_dns_record" "this" {
  for_each = local.records_by_key

  zone_id = data.cloudflare_zone.this.zone_id
  name    = each.value.name
  type    = each.value.type
  content = each.value.content
  ttl     = try(each.value.ttl, 1)
  proxied = try(each.value.proxied, false)
  comment = try(each.value.comment, null)
}

output "fritzbox_reminder" {
  value = <<-EOT

    ============================================================
    DO NOT FORGET: ADD ALL RECORDS TO FRITZBOX
    DNS REBIND PROTECTION ALLOWLIST OR LOCAL RESOLUTION BREAKS
    URL: http://fritz.box/#/network/settings/critical/dns-rebind-protection
    ============================================================
  EOT
}
