resource "random_password" "deploy_postgres" {
  count   = var.deploy_enabled && var.deploy_postgres_password == "" ? 1 : 0
  length  = 32
  special = false
}

resource "random_password" "deploy_jwt" {
  count   = var.deploy_enabled && var.deploy_jwt_secret == "" ? 1 : 0
  length  = 64
  special = false
}

locals {
  deploy_postgres_password = var.deploy_enabled ? (
    var.deploy_postgres_password != "" ? var.deploy_postgres_password : random_password.deploy_postgres[0].result
  ) : ""

  deploy_jwt_secret = var.deploy_enabled ? (
    var.deploy_jwt_secret != "" ? var.deploy_jwt_secret : random_password.deploy_jwt[0].result
  ) : ""

  deploy_env_content = var.deploy_enabled ? join("\n", [
    "POSTGRES_PASSWORD=${local.deploy_postgres_password}",
    "DATABASE_URL=postgres://studydrift:${urlencode(local.deploy_postgres_password)}@postgres:5432/studydrift?sslmode=disable",
    "JWT_SECRET=${local.deploy_jwt_secret}",
    "LEXTURES_SERVER_IMAGE=${var.deploy_server_image}",
    "LEXTURES_WEB_IMAGE=${var.deploy_web_image}",
    "",
  ]) : ""

  deploy_env_b64     = base64encode(local.deploy_env_content)
  deploy_compose_b64 = base64encode(file("${path.module}/../../../docker-compose.deploy.yml"))

  deploy_cloud_init_vars = {
    deploy_enabled           = var.deploy_enabled
    deploy_env_b64           = local.deploy_env_b64
    deploy_compose_b64       = local.deploy_compose_b64
    deploy_public_origin     = var.deploy_public_origin
    deploy_registry_host     = var.deploy_registry_host
    deploy_registry_username = var.deploy_registry_username
    deploy_registry_password = var.deploy_registry_password
  }

  mount_script_b64 = base64encode(templatefile("${path.module}/mount-db-volume.sh.tftpl", {
    db_volume_name = digitalocean_volume.data.name
  }))

  deploy_script_b64 = base64encode(templatefile(
    "${path.module}/../shared/small-vm-deploy.sh.tftpl",
    local.deploy_cloud_init_vars,
  ))

  bootstrap_script_b64 = base64encode(templatefile("${path.module}/../shared/small-vm-bootstrap.sh.tftpl", {
    deploy_enabled = var.deploy_enabled
  }))

  compose_service_b64 = base64encode(file("${path.module}/../shared/lextures-compose.service"))

  cloud_init_vars = merge(local.deploy_cloud_init_vars, {
    mount_script_b64     = local.mount_script_b64
    deploy_script_b64    = local.deploy_script_b64
    bootstrap_script_b64 = local.bootstrap_script_b64
    compose_service_b64  = local.compose_service_b64
  })
}

resource "terraform_data" "cloud_init_revision" {
  input = md5(jsonencode({
    mount     = local.mount_script_b64
    bootstrap = local.bootstrap_script_b64
    deploy    = local.deploy_script_b64
    service   = local.compose_service_b64
    enabled   = var.deploy_enabled
    server    = var.deploy_server_image
    web       = var.deploy_web_image
    origin    = var.deploy_public_origin
    registry  = "${var.deploy_registry_username}@${var.deploy_registry_host}"
  }))
}
