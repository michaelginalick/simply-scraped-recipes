provider "aws" {
  region  = var.region
  profile = "TERRAFORM_USER"
}

terraform {
  required_version = ">= 1.6.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.28"
    }
  }
}
