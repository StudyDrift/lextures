resource "tls_private_key" "app" {
  algorithm = "ED25519"
}

resource "digitalocean_ssh_key" "app" {
  name       = "${local.name_prefix}-ssh"
  public_key = tls_private_key.app.public_key_openssh
}

resource "digitalocean_volume" "data" {
  region = var.region
  name   = "${local.name_prefix}-data"
  size   = var.data_volume_size_gb
  tags   = local.common_tags
}

resource "digitalocean_droplet" "app" {
  name     = local.name_prefix
  region   = var.region
  size     = var.droplet_size
  image    = var.droplet_image
  ssh_keys = [digitalocean_ssh_key.app.id]
  tags     = local.common_tags

  backups = var.enable_droplet_backups

  volume_ids = [digitalocean_volume.data.id]

  lifecycle {
    replace_triggered_by = [
      terraform_data.cloud_init_revision,
    ]
  }

  user_data = templatefile("${path.module}/cloud-init.yaml.tftpl", local.cloud_init_vars)
}

resource "digitalocean_reserved_ip" "app" {
  region     = var.region
  droplet_id = digitalocean_droplet.app.id
}

resource "digitalocean_firewall" "app" {
  name = "${local.name_prefix}-fw"

  droplet_ids = [digitalocean_droplet.app.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "icmp"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}
