data "cloudinaryprovisioning_current_principal" "me" {}

data "cloudinaryprovisioning_role" "admin" {
  name            = "Admin"
  permission_type = "global"
}

resource "cloudinaryprovisioning_product_environment" "dev" {
  name = "backmarket-dev"
}

# Grant the provisioning key Admin access to the new product environment.
# This is required so that subsequent API calls (e.g. creating access keys)
# succeed within that product environment.
resource "cloudinaryprovisioning_principal_role_assignment" "provisioning_key_admin" {
  principal_type = data.cloudinaryprovisioning_current_principal.me.principal_type
  principal_id   = data.cloudinaryprovisioning_current_principal.me.principal_id
  role_id        = data.cloudinaryprovisioning_role.admin.id
  scope_type     = "prodenv"
  scope_id       = cloudinaryprovisioning_product_environment.dev.id
}
