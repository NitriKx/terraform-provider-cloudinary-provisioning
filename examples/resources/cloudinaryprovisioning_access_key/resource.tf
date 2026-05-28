resource "cloudinaryprovisioning_product_environment" "example" {
  name    = "my-project"
  enabled = true
}

resource "cloudinaryprovisioning_access_key" "example" {
  product_environment_id = cloudinaryprovisioning_product_environment.example.id
  name                   = "my-system-api-key"
  enabled                = true
}

output "api_key" {
  value = cloudinaryprovisioning_access_key.example.api_key
}

output "api_secret" {
  value     = cloudinaryprovisioning_access_key.example.api_secret
  sensitive = true
}
