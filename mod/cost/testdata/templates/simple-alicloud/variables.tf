variable "region" {
  description = "Alibaba Cloud region"
  type        = string
  default     = "cn-hangzhou"
}

variable "node_count" {
  description = "Number of instances to create"
  type        = number
  default     = 1
}

variable "instance_type" {
  description = "Instance type"
  type        = string
  default     = "ecs.g6.large"
}

variable "disk_size" {
  description = "System disk size in GB"
  type        = number
  default     = 40
}
