data "cloudinaryprovisioning_current_principal" "me" {}

output "provisioning_principal_id" {
  value = data.cloudinaryprovisioning_current_principal.me.principal_id
}

output "provisioning_principal_type" {
  value = data.cloudinaryprovisioning_current_principal.me.principal_type
}
