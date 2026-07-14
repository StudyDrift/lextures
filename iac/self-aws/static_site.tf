# CloudFront front door for the SPA.
# - web_image empty: S3 origin (static deploy via scripts/deploy-web.sh) + ALB for API paths
# - web_image set:   ALB origin for everything (ALB path-routes API vs nginx SPA)

locals {
  web_bucket_name = coalesce(
    var.web_bucket_name,
    "${local.name_prefix}-web-${data.aws_caller_identity.current.account_id}"
  )

  # Managed CloudFront policies (global; IDs are stable across accounts).
  cf_cache_optimized   = "658327ea-f89d-4fab-a63d-7e88639e58f6"
  cf_cache_disabled    = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
  cf_origin_all_viewer = "216adef6-5c7f-47e4-b989-5492eafa07d3" # headers/query/cookies → ALB
}

resource "aws_s3_bucket" "web" {
  # Retained whenever enable_static_site is true (even if CloudFront serves the web container).
  count = local.create_web_bucket ? 1 : 0

  bucket        = local.web_bucket_name
  force_destroy = var.web_bucket_force_destroy

  tags = {
    Name = local.web_bucket_name
  }
}

resource "aws_s3_bucket_public_access_block" "web" {
  count = local.create_web_bucket ? 1 : 0

  bucket = aws_s3_bucket.web[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "web" {
  count = local.create_web_bucket ? 1 : 0

  bucket = aws_s3_bucket.web[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_versioning" "web" {
  count = local.create_web_bucket ? 1 : 0

  bucket = aws_s3_bucket.web[0].id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_cloudfront_origin_access_control" "web" {
  count = local.create_web_bucket ? 1 : 0

  name                              = "${local.name_prefix}-web-oac"
  description                       = "OAC for static web bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "web" {
  count = var.enable_static_site ? 1 : 0

  enabled             = true
  is_ipv6_enabled     = true
  comment             = "${local.name_prefix} web"
  default_root_object = local.use_web_container ? "" : "index.html"
  price_class         = var.cloudfront_price_class
  aliases             = var.web_domain_names
  wait_for_deployment = false

  dynamic "origin" {
    for_each = local.create_web_bucket ? [1] : []
    content {
      domain_name              = aws_s3_bucket.web[0].bucket_regional_domain_name
      origin_id                = "s3-web"
      origin_access_control_id = aws_cloudfront_origin_access_control.web[0].id
    }
  }

  dynamic "origin" {
    for_each = var.enable_ecs ? [1] : []
    content {
      domain_name = aws_lb.main[0].dns_name
      origin_id   = "alb"

      custom_origin_config {
        http_port              = 80
        https_port             = 443
        origin_protocol_policy = "http-only"
        origin_ssl_protocols   = ["TLSv1.2"]
      }
    }
  }

  # SPA default: S3 when static-only; ALB (nginx web service) when web_image is set.
  default_cache_behavior {
    allowed_methods        = local.use_web_container ? ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"] : ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = local.use_web_container ? "alb" : "s3-web"
    viewer_protocol_policy = "redirect-to-https"
    compress               = true
    # Prefer origin Cache-Control when fronting nginx; disable cache for API-style methods on ALB SPA.
    cache_policy_id          = local.use_web_container ? local.cf_cache_disabled : local.cf_cache_optimized
    origin_request_policy_id = local.use_web_container ? local.cf_origin_all_viewer : null
  }

  # When SPA is on S3, proxy API + health + tus to Fargate via ALB (no cache).
  # When SPA is on Fargate, ALB already path-routes these — still pin no-cache + full viewer forward.
  dynamic "ordered_cache_behavior" {
    for_each = var.enable_ecs ? local.api_path_patterns : []
    content {
      path_pattern             = ordered_cache_behavior.value
      allowed_methods          = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
      cached_methods           = ["GET", "HEAD"]
      target_origin_id         = "alb"
      viewer_protocol_policy   = "redirect-to-https"
      compress                 = true
      cache_policy_id          = local.cf_cache_disabled
      origin_request_policy_id = local.cf_origin_all_viewer
    }
  }

  # Client-side routing for S3 origin: missing keys → index.html.
  # nginx try_files already rewrites SPA routes when the origin is the web container.
  dynamic "custom_error_response" {
    for_each = local.serve_spa_from_s3 ? [403, 404] : []
    content {
      error_code            = custom_error_response.value
      response_code         = 200
      response_page_path    = "/index.html"
      error_caching_min_ttl = 0
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  dynamic "viewer_certificate" {
    for_each = length(var.web_domain_names) == 0 ? [1] : []
    content {
      cloudfront_default_certificate = true
    }
  }

  dynamic "viewer_certificate" {
    for_each = length(var.web_domain_names) > 0 ? [1] : []
    content {
      acm_certificate_arn      = local.web_acm_certificate_arn_effective
      ssl_support_method       = "sni-only"
      minimum_protocol_version = "TLSv1.2_2021"
    }
  }

  # Wait for ACM issuance when Terraform manages the cert (CloudFront rejects PENDING_VALIDATION).
  depends_on = [aws_acm_certificate_validation.web]

  tags = {
    Name = "${local.name_prefix}-web"
  }
}

resource "aws_s3_bucket_policy" "web" {
  count = local.create_web_bucket ? 1 : 0

  bucket = aws_s3_bucket.web[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowCloudFrontServicePrincipal"
        Effect    = "Allow"
        Principal = { Service = "cloudfront.amazonaws.com" }
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.web[0].arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.web[0].arn
          }
        }
      }
    ]
  })
}
