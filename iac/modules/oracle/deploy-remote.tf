resource "null_resource" "app_deploy" {
  count = var.deploy_enabled ? 1 : 0

  triggers = {
    revision = terraform_data.deploy_revision.input
  }

  connection {
    type        = "ssh"
    host        = oci_core_public_ip.app.ip_address
    user        = "ubuntu"
    private_key = tls_private_key.app.private_key_openssh
    timeout     = "20m"
  }

  provisioner "file" {
    content     = local.deploy_script_content
    destination = "/tmp/lextures-deploy-app.sh"
  }

  provisioner "file" {
    content     = local.deploy_env_content
    destination = "/tmp/lextures.env"
  }

  provisioner "file" {
    content     = file("${path.root}/docker-compose.deploy.yml")
    destination = "/tmp/docker-compose.deploy.yml"
  }

  provisioner "file" {
    content     = local.compose_service_content
    destination = "/tmp/lextures.service"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo install -d -m 0755 /opt/lextures",
      "sudo install -m 0755 /tmp/lextures-deploy-app.sh /usr/local/bin/lextures-deploy-app.sh",
      "sudo install -m 0600 /tmp/lextures.env /opt/lextures/.env",
      "sudo install -m 0644 /tmp/docker-compose.deploy.yml /opt/lextures/docker-compose.deploy.yml",
      "sudo install -m 0644 /tmp/lextures.service /etc/systemd/system/lextures.service",
      "rm -f /tmp/lextures-deploy-app.sh /tmp/lextures.env /tmp/docker-compose.deploy.yml /tmp/lextures.service",
    ]
  }

  provisioner "remote-exec" {
    script = "${path.module}/../shared/terraform-wait-and-deploy.sh"
  }

  depends_on = [
    oci_core_instance.app,
    oci_core_public_ip.app,
    oci_core_volume_attachment.data,
  ]
}