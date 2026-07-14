# ACM certificate for CloudFront custom domains.
#
# CloudFront requires certificates in us-east-1 (provider alias), regardless of
# var.aws_region. When web_domain_names is set and web_acm_certificate_arn is
# empty, Terraform requests a DNS-validated cert and waits for issuance.
#
# Workflow when managing the cert here:
#   1. Set web_domain_names = ["beta.example.com"] (leave web_acm_certificate_arn empty)
#   2. terraform apply  (creates cert; may wait up to 45m for DNS validation)
#   3. While waiting (or after a targeted apply of aws_acm_certificate.web):
#        terraform output acm_dns_validation_records
#      Add those CNAMEs in your DNS (Cloudflare: DNS only / grey cloud)
#   4. After ISSUED, CloudFront picks up the cert on the same or next apply
#
# Or pass an existing us-east-1 ACM ARN via web_acm_certificate_arn to skip creation.

locals {
  web_acm_external = (
    var.web_acm_certificate_arn != null
    && var.web_acm_certificate_arn != ""
  )
  create_web_acm = length(var.web_domain_names) > 0 && !local.web_acm_external

  # Effective ARN for CloudFront (null when using the default *.cloudfront.net cert).
  web_acm_certificate_arn_effective = (
    length(var.web_domain_names) == 0
    ? null
    : (
      local.web_acm_external
      ? var.web_acm_certificate_arn
      : aws_acm_certificate_validation.web[0].certificate_arn
    )
  )
}

resource "aws_acm_certificate" "web" {
  count = local.create_web_acm ? 1 : 0

  provider = aws.us_east_1

  domain_name               = var.web_domain_names[0]
  subject_alternative_names = length(var.web_domain_names) > 1 ? slice(var.web_domain_names, 1, length(var.web_domain_names)) : []
  validation_method         = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name = "${local.name_prefix}-web-cloudfront"
  }
}

resource "aws_acm_certificate_validation" "web" {
  count = local.create_web_acm ? 1 : 0

  provider = aws.us_east_1

  certificate_arn = aws_acm_certificate.web[0].arn
  validation_record_fqdns = [
    for dvo in aws_acm_certificate.web[0].domain_validation_options : dvo.resource_record_name
  ]

  timeouts {
    create = "45m"
  }
}
