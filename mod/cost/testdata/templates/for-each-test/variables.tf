variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "security_groups" {
  description = "Security groups to create"
  type        = map(string)
  default = {
    web = "Web server security group"
    db  = "Database security group"
  }
}
