terraform {
  required_version = ">= 1.5"

  # Remote backend: configure in Terraform Cloud or copy backend.tf.example.
  # Example HCP workspace name: lextures-production-aws
  #
  # cloud {
  #   workspaces {
  #     name = "lextures-production-aws"
  #   }
  # }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80"
    }
  }
}
