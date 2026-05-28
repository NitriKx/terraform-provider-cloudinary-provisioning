resource "cloudinaryprovisioning_product_environment" "example" {
  name    = "my-project"
  enabled = true
  # deletion_protection defaults to true — set to false and apply before destroying
}

# Optionally specify a cloud_name (auto-generated if omitted)
resource "cloudinaryprovisioning_product_environment" "named" {
  name       = "my-project"
  cloud_name = "my-project-cloud"
  enabled    = true
}

# To destroy a protected environment:
#   1. Set deletion_protection = false and run terraform apply
#   2. Then run terraform destroy
resource "cloudinaryprovisioning_product_environment" "unprotected" {
  name                = "my-ephemeral-project"
  deletion_protection = false
}
