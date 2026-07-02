terraform {
  required_version = ">= 1.5"

  # Remote backend: configure in Terraform Cloud or copy backend.tf.example.
  # Example HCP workspace name: lextures-production-aws
  #
  cloud {
    organization = "Lextures"

    workspaces {
      name = "lextures-production"
    }
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80"
    }
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.34"
    }
    oci = {
      source  = "oracle/oci"
      version = "~> 6.30"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    statuspage = {
      source  = "winebarrel/statuspage"
      version = "~> 0.3"
    }
  }
}
