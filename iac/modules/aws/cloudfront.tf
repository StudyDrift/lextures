# CloudFront CDN for course file uploads (plan 17.5 FR-5).
# Presigned URLs are rewritten to this distribution when STORAGE_CDN_BASE_URL is set.

resource "aws_cloudfront_origin_access_control" "course_files" {
  name                              = "${local.course_files_bucket_name}-oac"
  description                       = "OAC for ${local.course_files_bucket_name}"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "course_files" {
  enabled             = true
  comment             = "${var.project_name} ${var.environment} course files CDN"
  default_root_object = ""
  price_class         = "PriceClass_100"

  origin {
    domain_name              = aws_s3_bucket.course_files.bucket_regional_domain_name
    origin_id                = "s3-course-files"
    origin_access_control_id = aws_cloudfront_origin_access_control.course_files.id
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "s3-course-files"
    viewer_protocol_policy = "redirect-to-https"
    compress               = true

    forwarded_values {
      query_string = true
      cookies {
        forward = "none"
      }
    }

    min_ttl     = 0
    default_ttl = 3600
    max_ttl     = 86400
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  tags = merge(local.common_tags, {
    Name = "${local.course_files_bucket_name}-cdn"
  })
}

resource "aws_s3_bucket_policy" "course_files_cdn" {
  bucket = aws_s3_bucket.course_files.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowCloudFrontServicePrincipal"
        Effect    = "Allow"
        Principal = { Service = "cloudfront.amazonaws.com" }
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.course_files.arn}/*"
        Condition = {
          StringEquals = {
            "AWS:SourceArn" = aws_cloudfront_distribution.course_files.arn
          }
        }
      }
    ]
  })
}
