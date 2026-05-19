# Terraform Provider: Cloudinary Provisioning

Manages account-level Cloudinary resources: product environments, access keys, custom policies, and role assignments.

For product-environment-level resources (folders, assets), see the companion
[cloudinary provider](https://github.com/NitriKx/terraform-provider-cloudinary).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 or [OpenTofu](https://opentofu.org/) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23 (to build from source)

## Installation

### Terraform / OpenTofu Registry

```hcl
terraform {
  required_providers {
    cloudinaryprovisioning = {
      source  = "NitriKx/cloudinary-provisioning"
      version = "~> 0.1"
    }
  }
}
```

### OpenTofu: install directly from GitHub (no registry required)

```hcl
terraform {
  required_providers {
    cloudinaryprovisioning = {
      source  = "NitriKx/cloudinary-provisioning"
      version = "~> 0.1"
    }
  }
}
```

Add to `~/.terraformrc` (or `~/.tofurc`):

```hcl
provider_installation {
  direct {
    include = ["registry.opentofu.org/NitriKx/cloudinary-provisioning"]
  }
}
```

## Authentication

The provider authenticates against the Cloudinary Provisioning API. Credentials can be supplied in two ways.

### Via `CLOUDINARY_ACCOUNT_URL` (recommended)

```bash
export CLOUDINARY_ACCOUNT_URL="account://API_KEY:API_SECRET@ACCOUNT_ID"
```

```hcl
provider "cloudinaryprovisioning" {}
```

### Via explicit attributes

```hcl
provider "cloudinaryprovisioning" {
  account_id = "your-account-id"
  api_key    = "your-api-key"
  api_secret = "your-api-secret"
}
```

Environment variable fallbacks: `CLOUDINARY_ACCOUNT_ID`, `CLOUDINARY_API_KEY`, `CLOUDINARY_API_SECRET`.

## Resources

### `cloudinaryprovisioning_product_environment`

Manages a Cloudinary product environment (sub-account).

```hcl
resource "cloudinaryprovisioning_product_environment" "example" {
  name    = "my-project"
  enabled = true
}
```

**Arguments:**
- `name` (Required) - Display name of the product environment.
- `cloud_name` (Optional, Computed) - Cloudinary cloud name. Auto-generated if not provided.
- `enabled` (Optional, Computed) - Whether the environment is enabled. Defaults to `true`.
- `base_sub_account_id` (Optional) - ID of an existing environment to copy settings from. Forces replacement.

**Attributes:**
- `id` - The unique identifier assigned by Cloudinary.
- `created_at`, `updated_at` - RFC3339 timestamps.

**Import:**
```bash
terraform import cloudinaryprovisioning_product_environment.example <id>
```

### `cloudinaryprovisioning_access_key`

Manages an API access key for a product environment.

```hcl
resource "cloudinaryprovisioning_access_key" "example" {
  product_environment_id = cloudinaryprovisioning_product_environment.example.id
  name                   = "my-system-key"
  enabled                = true
}
```

**Arguments:**
- `product_environment_id` (Required) - The product environment ID. Forces replacement on change.
- `name` (Optional, Computed) - Display name for the key.
- `enabled` (Optional, Computed) - Whether the key is enabled. Defaults to `true`.

**Attributes:**
- `api_key` - The API key identifier.
- `api_secret` - The API secret (sensitive, only available on creation, preserved in state).
- `created_at`, `updated_at` - RFC3339 timestamps.

**Import:**
```bash
terraform import cloudinaryprovisioning_access_key.example <product_environment_id>/<api_key>
```

### `cloudinaryprovisioning_custom_policy`

Manages a custom Cedar permissions policy.

```hcl
resource "cloudinaryprovisioning_custom_policy" "example" {
  name       = "my-policy"
  scope_type = "prodenv"
  scope_id   = cloudinaryprovisioning_product_environment.example.id
  enabled    = true

  policy_statement = <<-EOT
    permit(
      principal == Cloudinary::APIKey::"${cloudinaryprovisioning_access_key.example.api_key}",
      action in [Cloudinary::Action::"create", Cloudinary::Action::"read"],
      resource is Cloudinary::Asset
    );
  EOT
}
```

**Arguments:**
- `name` (Required) - Display name of the policy.
- `policy_statement` (Required) - Cedar policy statement.
- `scope_type` (Required) - `"account"` or `"prodenv"`.
- `scope_id` (Optional, Computed) - Product environment ID. Required when `scope_type` is `"prodenv"`.
- `enabled` (Optional, Computed) - Whether the policy is enabled. Defaults to `true`.
- `description` (Optional, Computed) - Human-readable description.

**Attributes:**
- `id` - The unique identifier assigned by Cloudinary.
- `created_at`, `updated_at` - Timestamps.

**Import:**
```bash
terraform import cloudinaryprovisioning_custom_policy.example <id>
```

### `cloudinaryprovisioning_principal_role_assignment`

Assigns a role to a principal (user, group, API key, or provisioning key) within a given scope.
All attributes are immutable: any change forces resource replacement.

A common use case is granting a provisioning key Admin access to a newly created product environment,
which is required before access keys can be created within that environment.

```hcl
data "cloudinaryprovisioning_role" "admin" {
  name            = "Admin"
  permission_type = "global"
}

resource "cloudinaryprovisioning_principal_role_assignment" "provisioning_key_admin" {
  principal_type = "provisioningKey"
  principal_id   = var.provisioning_api_key_id
  role_id        = data.cloudinaryprovisioning_role.admin.id
  scope_type     = "prodenv"
  scope_id       = cloudinaryprovisioning_product_environment.example.id
}
```

**Arguments:**
- `principal_type` (Required) - Type of principal: `"user"`, `"group"`, `"apiKey"`, or `"provisioningKey"`. Forces replacement.
- `principal_id` (Required) - The unique identifier of the principal. Forces replacement.
- `role_id` (Required) - The role to assign (use the `cloudinaryprovisioning_role` data source to look up by name). Forces replacement.
- `scope_type` (Required) - `"account"` or `"prodenv"`. Forces replacement.
- `scope_id` (Optional) - Product environment ID. Required when `scope_type` is `"prodenv"`. Forces replacement.

**Attributes:**
- `id` - Composite identifier: `{principal_type}/{principal_id}/{role_id}/{scope_type}/{scope_id}`.

**Import:**
```bash
terraform import cloudinaryprovisioning_principal_role_assignment.example \
  provisioningKey/<principal_id>/<role_id>/prodenv/<scope_id>
```

## Data Sources

### `cloudinaryprovisioning_product_environment`

Reads an existing product environment by `id` or `cloud_name`.

```hcl
data "cloudinaryprovisioning_product_environment" "current" {
  cloud_name = "my-project-cloud"
}
```

**Arguments (one required):**
- `id` (Optional, Computed) - Look up by ID.
- `cloud_name` (Optional, Computed) - Look up by cloud name.

**Attributes:** `name`, `enabled`, `created_at`, `updated_at`.

### `cloudinaryprovisioning_current_principal`

Returns the identity of the currently authenticated principal (the credentials used to configure
the provider). Use `principal_id` and `principal_type` directly in role assignments.
No API call is made — the values come from the provider configuration.

```hcl
data "cloudinaryprovisioning_current_principal" "me" {}

resource "cloudinaryprovisioning_principal_role_assignment" "admin" {
  principal_type = data.cloudinaryprovisioning_current_principal.me.principal_type
  principal_id   = data.cloudinaryprovisioning_current_principal.me.principal_id
  role_id        = data.cloudinaryprovisioning_role.admin.id
  scope_type     = "prodenv"
  scope_id       = cloudinaryprovisioning_product_environment.dev.id
}
```

**Attributes:**
- `principal_id` - The unique identifier of the current principal.
- `principal_type` - The type of the current principal (currently always `"provisioningKey"`).
- `account_id` - The Cloudinary account ID the provider is configured for.

### `cloudinaryprovisioning_role`

Looks up a Cloudinary role by name and permission type. Use this to retrieve the role ID
needed for `cloudinaryprovisioning_principal_role_assignment`.

```hcl
data "cloudinaryprovisioning_role" "admin" {
  name            = "Admin"
  permission_type = "global"
}

output "admin_role_id" {
  value = data.cloudinaryprovisioning_role.admin.id
}
```

**Arguments:**
- `name` (Required) - The role name to look up (e.g. `"Admin"`).
- `permission_type` (Required) - `"global"` (role applies across all contexts within its scope, e.g. all folders in a product environment) or `"content"` (role applies to a specific content instance such as a particular folder or collection). Most built-in roles like `"Admin"` are `"global"`.

**Attributes:** `id`, `description`, `management_type`, `scope_type`.

## Using both providers together

```hcl
terraform {
  required_providers {
    cloudinary = {
      source  = "NitriKx/cloudinary"
      version = "~> 0.1"
    }
    cloudinaryprovisioning = {
      source  = "NitriKx/cloudinary-provisioning"
      version = "~> 0.1"
    }
  }
}

provider "cloudinary" {}             # reads CLOUDINARY_URL
provider "cloudinaryprovisioning" {} # reads CLOUDINARY_ACCOUNT_URL

data "cloudinaryprovisioning_product_environment" "current" {
  cloud_name = "my-cloud"
}

resource "cloudinary_folder" "assets" {
  path = "my-project/images"
}

resource "cloudinaryprovisioning_access_key" "system" {
  product_environment_id = data.cloudinaryprovisioning_product_environment.current.id
  name                   = "system-key"
}

resource "cloudinaryprovisioning_custom_policy" "restrict" {
  name       = "restrict-to-folder"
  scope_type = "prodenv"
  scope_id   = data.cloudinaryprovisioning_product_environment.current.id

  policy_statement = <<-EOT
    permit(
      principal == Cloudinary::APIKey::"${cloudinaryprovisioning_access_key.system.api_key}",
      action in [Cloudinary::Action::"create", Cloudinary::Action::"read"],
      resource is Cloudinary::Asset
    ) when {
      resource.ancestor_ids.contains("${cloudinary_folder.assets.external_id}")
    };
  EOT
}
```

## Development

```bash
# Build
task build

# Install locally for testing
task install

# Run unit tests
task test

# Run linter
task lint

# Generate docs
task docs
```

For local dev overrides, add to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "NitriKx/cloudinary-provisioning" = "/path/to/terraform-provider-cloudinary-provisioning"
  }
  direct {}
}
```
