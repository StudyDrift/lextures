data "aws_iam_policy_document" "course_files_bucket" {
  statement {
    sid    = "ListBucket"
    effect = "Allow"
    actions = [
      "s3:ListBucket",
    ]
    resources = [
      aws_s3_bucket.course_files.arn,
    ]
  }

  statement {
    sid    = "ObjectRW"
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = [
      "${aws_s3_bucket.course_files.arn}/*",
    ]
  }
}

module "irsa_course_files" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.48"

  role_name = "${local.name_prefix}-course-files"

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["lextures:api"]
    }
  }

  role_policy_arns = {
    course_files = aws_iam_policy.course_files.arn
  }

  tags = local.common_tags
}

resource "aws_iam_policy" "course_files" {
  name_prefix = "${local.name_prefix}-course-files-"
  description = "Read/write course files in the production S3 bucket"
  policy      = data.aws_iam_policy_document.course_files_bucket.json

  tags = local.common_tags
}
