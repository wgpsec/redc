variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "node_count" {
  description = "Number of instances to create"
  type        = number
  default     = 1
}

variable "instance_type" {
  description = "Instance type"
  type        = string
  default     = "t2.micro"
}

variable "disk_size" {
  description = "Root volume size in GB"
  type        = number
  default     = 20
}
