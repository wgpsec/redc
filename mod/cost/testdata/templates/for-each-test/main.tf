terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# Resource with for_each using a map
resource "aws_instance" "servers" {
  for_each = {
    web = "t2.micro"
    api = "t2.small"
    db  = "t2.medium"
  }
  
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = each.value
  
  tags = {
    Name = each.key
  }
}

# Resource with for_each using a set
resource "aws_s3_bucket" "buckets" {
  for_each = toset(["logs", "data", "backups"])
  
  bucket = "my-${each.key}-bucket"
}

# Resource with for_each using a variable
resource "aws_security_group" "groups" {
  for_each = var.security_groups
  
  name        = each.key
  description = each.value
}
