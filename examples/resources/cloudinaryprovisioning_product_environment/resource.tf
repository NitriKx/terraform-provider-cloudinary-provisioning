resource "cloudinaryprovisioning_product_environment" "example" {
  name    = "my-project"
  enabled = true
}

# Optionally specify a cloud_name (auto-generated if omitted)
resource "cloudinaryprovisioning_product_environment" "named" {
  name       = "my-project"
  cloud_name = "my-project-cloud"
  enabled    = true
}
