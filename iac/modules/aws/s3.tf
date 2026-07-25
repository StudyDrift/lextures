resource "aws_s3_bucket" "course_files" {
  bucket = local.course_files_bucket_name

  force_destroy = var.course_files_bucket_force_destroy

  tags = merge(local.common_tags, {
    Name = local.course_files_bucket_name
  })
}

resource "aws_s3_bucket_versioning" "course_files" {
  bucket = aws_s3_bucket.course_files.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "course_files" {
  bucket = aws_s3_bucket.course_files.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_public_access_block" "course_files" {
  bucket = aws_s3_bucket.course_files.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Browser direct uploads use presigned PUT URLs. Without CORS the SPA origin
# fails the OPTIONS preflight ("No 'Access-Control-Allow-Origin' header").
resource "aws_s3_bucket_cors_configuration" "course_files" {
  count = length(var.course_files_cors_allowed_origins) > 0 ? 1 : 0

  bucket = aws_s3_bucket.course_files.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "PUT", "HEAD"]
    allowed_origins = var.course_files_cors_allowed_origins
    expose_headers  = ["ETag", "Content-Length", "Content-Type"]
    max_age_seconds = 3600
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "course_files" {
  bucket = aws_s3_bucket.course_files.id

  rule {
    id     = "abort-incomplete-multipart"
    status = "Enabled"

    filter {}

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}
