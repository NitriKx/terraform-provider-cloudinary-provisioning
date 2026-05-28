# Look up by cloud_name
data "cloudinaryprovisioning_product_environment" "by_cloud_name" {
  cloud_name = "my-project-cloud"
}

# Look up by ID
data "cloudinaryprovisioning_product_environment" "by_id" {
  id = "abc123"
}

output "product_environment_id" {
  value = data.cloudinaryprovisioning_product_environment.by_cloud_name.id
}
