# Using CLOUDINARY_ACCOUNT_URL environment variable (recommended)
# export CLOUDINARY_ACCOUNT_URL="account://API_KEY:API_SECRET@ACCOUNT_ID"
provider "cloudinaryprovisioning" {}

# Or using explicit credentials
provider "cloudinaryprovisioning" {
  account_id = "your-account-id"
  api_key    = "your-api-key"
  api_secret = "your-api-secret"
}
