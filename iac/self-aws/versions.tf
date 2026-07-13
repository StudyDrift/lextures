terraform {
  required_version = ">= 1.5"

  cloud {
    organization = "Lextures"

    workspaces {
      name = "lextures-self-aws-production"
    }
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}
