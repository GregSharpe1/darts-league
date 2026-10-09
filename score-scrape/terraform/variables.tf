variable "region" {
  type        = string
  description = "Explicit target AWS region. Select the account through your own AWS credential chain."
}

variable "name_prefix" {
  type        = string
  description = "Unique standalone deployment name; never reuse CloudFormation-owned names."
  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{0,39}$", var.name_prefix))
    error_message = "Use 1-40 lowercase letters, digits or hyphens, starting with a letter."
  }
}

variable "allowed_origin" {
  type    = string
  default = "https://play.autodarts.com"
  validation {
    condition     = can(regex("^https://[a-zA-Z0-9.-]+(:[0-9]+)?$", var.allowed_origin))
    error_message = "Use one exact HTTPS origin without wildcard, path or trailing slash."
  }
}

variable "alarm_action_arns" {
  type        = list(string)
  default     = []
  description = "Operator-managed alert destinations. Empty means alarms have no notification routing."
}

variable "log_retention_days" {
  type    = number
  default = 30
  validation {
    condition     = contains([1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1096, 1827, 2192, 2557, 2922, 3288, 3653], var.log_retention_days)
    error_message = "Select a supported finite CloudWatch log retention period."
  }
}
