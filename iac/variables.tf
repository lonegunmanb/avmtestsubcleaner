variable "client_id" {
  type = string
}

variable "client_secret" {
  sensitive = true
  type      = string
}