terraform {
  required_providers {
    alicloud = {
      source  = "aliyun/alicloud"
       version = "1.241.0"
    }
  }
}

provider "alicloud" {
  profile = "cloud-tool"
  region  = "cn-beijing"
}