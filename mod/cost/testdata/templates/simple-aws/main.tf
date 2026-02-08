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

resource "aws_instance" "web" {
  count         = var.node_count
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = var.instance_type
  
  root_block_device {
    volume_size = var.disk_size
  }
  
  tags = {
    Name = "web-${count.index}"
  }
}
