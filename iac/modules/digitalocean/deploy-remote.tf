resource "null_resource" "app_deploy" {
  count = var.deploy_enabled ? 1 : 0

  triggers = {
    revision = terraform_data.cloud_init_revision.input
  }

  connection {
    type        = "ssh"
    host        = digitalocean_reserved_ip.app.ip_address
    user        = "root"
    private_key = tls_private_key.app.private_key_openssh
    timeout     = "20m"
  }

  provisioner "remote-exec" {
    script = "${path.module}/../shared/terraform-wait-and-deploy.sh"
  }

  depends_on = [
    digitalocean_droplet.app,
    digitalocean_reserved_ip.app,
  ]
}
