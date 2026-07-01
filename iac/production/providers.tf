locals {
  use_aws          = var.cloud_provider == "aws"
  use_digitalocean = var.cloud_provider == "digitalocean"
  use_oracle       = var.cloud_provider == "oracle"

  oci_stub = {
    tenancy_ocid     = "ocid1.tenancy.oc1..aaaaaaaaxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    user_ocid        = "ocid1.user.oc1..aaaaaaaaxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    fingerprint      = "8a:be:56:a2:f7:07:5c:ea:04:4d:31:d0:4f:14:6e:d9"
    private_key_path = "${path.module}/stub-oci.pem"
  }

  oci_explicit_auth = (
    var.oci_tenancy_ocid != "" &&
    var.oci_user_ocid != "" &&
    var.oci_fingerprint != "" &&
    (var.oci_private_key_path != "" || var.oci_private_key != "")
  )

  oci_config_file_present = fileexists(pathexpand("~/.oci/config"))
  oci_use_config_file     = local.use_oracle && var.oci_auth_method == "config_file" && local.oci_config_file_present
  oci_use_tfvars_auth     = local.use_oracle && var.oci_auth_method == "tfvars" && local.oci_explicit_auth

  oracle_module_enabled = local.oci_use_tfvars_auth || local.oci_use_config_file

  oci_tenancy_ocid = local.use_oracle ? (
    local.oci_use_config_file ? null : (
      local.oci_explicit_auth ? var.oci_tenancy_ocid : local.oci_stub.tenancy_ocid
    )
  ) : local.oci_stub.tenancy_ocid
  oci_user_ocid = local.use_oracle ? (
    local.oci_use_config_file ? null : (
      local.oci_explicit_auth ? var.oci_user_ocid : local.oci_stub.user_ocid
    )
  ) : local.oci_stub.user_ocid
  oci_fingerprint = local.use_oracle ? (
    local.oci_use_config_file ? null : (
      local.oci_explicit_auth ? var.oci_fingerprint : local.oci_stub.fingerprint
    )
  ) : local.oci_stub.fingerprint
  oci_private_key_path = local.use_oracle ? (
    local.oci_use_config_file || var.oci_private_key != "" ? null : (
      local.oci_explicit_auth ? var.oci_private_key_path : local.oci_stub.private_key_path
    )
  ) : local.oci_stub.private_key_path
  oci_private_key    = local.use_oracle && local.oci_use_tfvars_auth && var.oci_private_key != "" ? var.oci_private_key : null
  oci_config_profile = local.oci_use_config_file ? var.oci_config_profile : null
}

provider "aws" {
  region = var.aws_region

  # Avoid credential lookup when another cloud is selected (module.aws count = 0).
  access_key                  = local.use_aws ? null : "unused"
  secret_key                  = local.use_aws ? null : "unused"
  skip_credentials_validation = !local.use_aws
  skip_metadata_api_check     = !local.use_aws
  skip_requesting_account_id  = !local.use_aws

  dynamic "default_tags" {
    for_each = local.use_aws ? [1] : []
    content {
      tags = {
        Project     = var.project_name
        Environment = var.environment
        Cloud       = "aws"
        ManagedBy   = "terraform"
      }
    }
  }
}

provider "digitalocean" {
  # Token is only required when the DigitalOcean module is active.
  token = local.use_digitalocean ? (var.digitalocean_token != "" ? var.digitalocean_token : null) : "unused"
}

provider "oci" {
  region = var.oci_region

  config_file_profile = local.oci_config_profile

  # When Oracle is not selected, use throwaway credentials (no OCI API calls are made).
  # When Oracle is selected, use oci_* tfvars (default) or ~/.oci/config when oci_auth_method = "config_file".
  tenancy_ocid     = local.oci_tenancy_ocid
  user_ocid        = local.oci_user_ocid
  fingerprint      = local.oci_fingerprint
  private_key_path = local.oci_private_key_path
  private_key      = local.oci_private_key
}
