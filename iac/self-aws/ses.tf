# Amazon SES for transactional email (password reset, magic links, COPPA, etc.).
# The API uses EMAIL_PROVIDER=ses and the default AWS credential chain (ECS task role).

locals {
  ses_enabled     = var.enable_ses && var.ses_domain != ""
  ses_from_email  = var.ses_from_email != "" ? var.ses_from_email : (local.ses_enabled ? "no-reply@${var.ses_domain}" : "")
  ses_config_name = var.ses_configuration_set_name != "" ? var.ses_configuration_set_name : "${local.name_prefix}-default"
}

resource "aws_sesv2_email_identity" "domain" {
  count = local.ses_enabled ? 1 : 0

  email_identity = var.ses_domain

  tags = {
    Name = "${local.name_prefix}-ses-domain"
  }
}

# Easy DKIM (RSA 2048) — publish the three CNAME records from outputs.
resource "aws_sesv2_email_identity_mail_from_attributes" "domain" {
  count = local.ses_enabled && var.ses_mail_from_subdomain != "" ? 1 : 0

  email_identity         = aws_sesv2_email_identity.domain[0].email_identity
  mail_from_domain       = "${var.ses_mail_from_subdomain}.${var.ses_domain}"
  behavior_on_mx_failure = "USE_DEFAULT_VALUE"
}

resource "aws_sesv2_configuration_set" "main" {
  count = local.ses_enabled ? 1 : 0

  configuration_set_name = local.ses_config_name

  tags = {
    Name = "${local.name_prefix}-ses-config"
  }
}

# Optional: verify a specific From address (useful while domain DKIM is pending).
resource "aws_sesv2_email_identity" "from_address" {
  count = local.ses_enabled && var.ses_verify_from_email && local.ses_from_email != "" ? 1 : 0

  email_identity = local.ses_from_email

  tags = {
    Name = "${local.name_prefix}-ses-from"
  }
}
