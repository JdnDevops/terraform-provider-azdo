terraform {
  required_providers {
    azdo = {
      source = "registry.terraform.io/jdn/azdo"
    }
  }
}
 
provider "azdo" {}
 
data "azdo_identities" "example" {}
output "identities_id" {
  value = data.azdo_identities.example
}

data "azdo_identity" "example" {
  display_name = "[DefaultCollection]\\Production"
}

output "identity_id" {
  value = data.azdo_identity.example
}
 