terraform {
  required_providers {
    alicloud = {
      source  = "aliyun/alicloud"
      version = "~> 1.0"
    }
  }
}

provider "alicloud" {
  region = var.region
}

resource "alicloud_instance" "web" {
  count             = var.node_count
  instance_type     = var.instance_type
  image_id          = "ubuntu_20_04_x64_20G_alibase_20210420.vhd"
  system_disk_size  = var.disk_size
  availability_zone = "${var.region}a"
  
  tags = {
    Name = "web-${count.index}"
  }
}
