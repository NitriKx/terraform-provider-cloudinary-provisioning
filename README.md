# Terraform Provider: Cloudinary Provisioning

Manages account-level Cloudinary resources: product environments, access keys, and custom policies.

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
