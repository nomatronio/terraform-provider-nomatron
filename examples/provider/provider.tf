terraform {
  required_providers {
    nomatron = {
      source  = "nomatronio/nomatron"
      version = "~> 0.6"
    }
  }
}

provider "nomatron" {
  address = "http://localhost:4649"
  token   = var.service_account_token
}
