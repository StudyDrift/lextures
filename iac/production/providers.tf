provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = var.project_name
      Environment = var.environment
      Cloud       = "aws"
      ManagedBy   = "terraform"
    }
  }
}
