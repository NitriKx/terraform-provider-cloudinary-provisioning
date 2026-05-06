data "cloudinaryprovisioning_product_environment" "example" {
  cloud_name = "my-project-cloud"
}

resource "cloudinaryprovisioning_access_key" "example" {
  product_environment_id = data.cloudinaryprovisioning_product_environment.example.id
  name                   = "my-system-api-key"
}

resource "cloudinaryprovisioning_custom_policy" "example" {
  name       = "my-project-policy"
  scope_type = "prodenv"
  scope_id   = data.cloudinaryprovisioning_product_environment.example.id
  enabled    = true

  policy_statement = <<-EOT
    permit(
      principal == Cloudinary::APIKey::"${cloudinaryprovisioning_access_key.example.api_key}",
      action in [Cloudinary::Action::"create", Cloudinary::Action::"update", Cloudinary::Action::"read", Cloudinary::Action::"delete"],
      resource is Cloudinary::Asset
    );
  EOT
}
