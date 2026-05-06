package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/cloudinary/account-provisioning-go/cldprovisioning"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/components"

	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/resources/access_key"
	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/resources/custom_policy"
	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/resources/product_environment"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &CloudinaryProvisioningProvider{}

// CloudinaryProvisioningProvider implements the Terraform provider for Cloudinary Provisioning API.
type CloudinaryProvisioningProvider struct {
	version string
}

// New creates a new CloudinaryProvisioningProvider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CloudinaryProvisioningProvider{version: version}
	}
}

// providerModel holds the provider configuration read from HCL.
type providerModel struct {
	AccountURL types.String `tfsdk:"account_url"`
	AccountID  types.String `tfsdk:"account_id"`
	APIKey     types.String `tfsdk:"api_key"`
	APISecret  types.String `tfsdk:"api_secret"`
}

func (p *CloudinaryProvisioningProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cloudinaryprovisioning"
	resp.Version = p.version
}

func (p *CloudinaryProvisioningProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Cloudinary Provisioning provider manages account-level Cloudinary resources " +
			"such as product environments, access keys, and custom policies.",
		Attributes: map[string]schema.Attribute{
			"account_url": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Cloudinary account URL in the format account://API_KEY:API_SECRET@ACCOUNT_ID. " +
					"When set, takes precedence over account_id, api_key, and api_secret. " +
					"Falls back to the CLOUDINARY_ACCOUNT_URL environment variable.",
			},
			"account_id": schema.StringAttribute{
				Optional: true,
				Description: "Cloudinary account ID. " +
					"Falls back to the CLOUDINARY_ACCOUNT_ID environment variable.",
			},
			"api_key": schema.StringAttribute{
				Optional: true,
				Description: "Cloudinary Provisioning API key. " +
					"Falls back to the CLOUDINARY_API_KEY environment variable.",
			},
			"api_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Cloudinary Provisioning API secret. " +
					"Falls back to the CLOUDINARY_API_SECRET environment variable.",
			},
		},
	}
}

func (p *CloudinaryProvisioningProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountURL := resolveValue(data.AccountURL, "CLOUDINARY_ACCOUNT_URL")
	accountID := resolveValue(data.AccountID, "CLOUDINARY_ACCOUNT_ID")
	apiKey := resolveValue(data.APIKey, "CLOUDINARY_API_KEY")
	apiSecret := resolveValue(data.APISecret, "CLOUDINARY_API_SECRET")

	var client *cldprovisioning.CldProvisioning

	switch {
	case accountURL != "":
		parsedAccountID, parsedKey, parsedSecret, parseErr := parseAccountURL(accountURL)
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid account_url", fmt.Sprintf("Failed to parse account_url: %s", parseErr))
			return
		}
		client = cldprovisioning.New(
			cldprovisioning.WithAccountID(parsedAccountID),
			cldprovisioning.WithSecurity(components.Security{
				ProvisioningAPIKey:    &parsedKey,
				ProvisioningAPISecret: &parsedSecret,
			}),
		)

	case accountID != "" && apiKey != "" && apiSecret != "":
		client = cldprovisioning.New(
			cldprovisioning.WithAccountID(accountID),
			cldprovisioning.WithSecurity(components.Security{
				ProvisioningAPIKey:    &apiKey,
				ProvisioningAPISecret: &apiSecret,
			}),
		)

	default:
		resp.Diagnostics.AddError(
			"Missing Cloudinary Provisioning credentials",
			"Provide either account_url (or CLOUDINARY_ACCOUNT_URL env var), "+
				"or all of account_id, api_key, and api_secret "+
				"(or their CLOUDINARY_ACCOUNT_ID / CLOUDINARY_API_KEY / CLOUDINARY_API_SECRET equivalents).",
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *CloudinaryProvisioningProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		product_environment.NewResource,
		access_key.NewResource,
		custom_policy.NewResource,
	}
}

func (p *CloudinaryProvisioningProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		product_environment.NewDataSource,
	}
}

// resolveValue returns the string value of a types.String attribute if set,
// otherwise falls back to the named environment variable.
func resolveValue(attr types.String, envVar string) string {
	if !attr.IsNull() && !attr.IsUnknown() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	return os.Getenv(envVar)
}

// parseAccountURL parses a Cloudinary account URL in the form:
//
//	account://KEY:SECRET@ACCOUNT_ID
//
// Returns (accountID, key, secret, error).
func parseAccountURL(raw string) (accountID, key, secret string, err error) {
	normalized := raw
	if !strings.Contains(normalized, "://") {
		normalized = "account://" + normalized
	}

	u, err := url.Parse(normalized)
	if err != nil {
		return "", "", "", fmt.Errorf("cannot parse URL: %w", err)
	}

	if u.User == nil {
		return "", "", "", fmt.Errorf("URL must contain credentials (KEY:SECRET@ACCOUNT_ID)")
	}

	k := u.User.Username()
	s, _ := u.User.Password()
	id := u.Hostname()

	if k == "" || s == "" || id == "" {
		return "", "", "", fmt.Errorf("URL must contain KEY, SECRET and ACCOUNT_ID")
	}

	return id, k, s, nil
}
