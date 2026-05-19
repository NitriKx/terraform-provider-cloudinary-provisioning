package product_environment

import (
	"context"
	"fmt"

	"github.com/cloudinary/account-provisioning-go/cldprovisioning"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/operations"

	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/providerdata"
	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &productEnvironmentDataSource{}

// NewDataSource returns a new cloudinaryprovisioning_product_environment data source.
func NewDataSource() datasource.DataSource {
	return &productEnvironmentDataSource{}
}

type productEnvironmentDataSource struct {
	client *cldprovisioning.CldProvisioning
}

type productEnvironmentDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	CloudName types.String `tfsdk:"cloud_name"`
	Name      types.String `tfsdk:"name"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *productEnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_environment"
}

func (d *productEnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads information about an existing Cloudinary product environment (sub-account). " +
			"Look up by id or by cloud_name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier of the product environment. When set, takes precedence over cloud_name.",
			},
			"cloud_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Cloudinary cloud name for this product environment. Used to look up the environment when id is not set.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The display name of the product environment.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the product environment is enabled.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the product environment was created (RFC3339).",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the product environment was last updated (RFC3339).",
			},
		},
	}
}

func (d *productEnvironmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *providerdata.ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	d.client = pd.Client
}

func (d *productEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data productEnvironmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		// Look up by explicit ID.
		result, err := d.client.ProductEnvironments.Get(ctx, data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading product environment", err.Error())
			return
		}
		if result == nil {
			resp.Diagnostics.AddError("Product environment not found",
				fmt.Sprintf("No product environment with id %q was found.", data.ID.ValueString()))
			return
		}
		data.ID = types.StringValue(util.DerefString(result.ID))
		data.Name = types.StringValue(util.DerefString(result.Name))
		data.CloudName = types.StringValue(util.DerefString(result.CloudName))
		data.Enabled = types.BoolValue(util.DerefBool(result.Enabled))
		if result.CreatedAt != nil {
			data.CreatedAt = types.StringValue(result.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
		} else {
			data.CreatedAt = types.StringValue("")
		}
		if result.UpdatedAt != nil {
			data.UpdatedAt = types.StringValue(result.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
		} else {
			data.UpdatedAt = types.StringValue("")
		}
	} else if !data.CloudName.IsNull() && data.CloudName.ValueString() != "" {
		// Look up by cloud_name via List + filter.
		cloudName := data.CloudName.ValueString()
		list, err := d.client.ProductEnvironments.List(ctx, &operations.GetProductEnvironmentsRequest{})
		if err != nil {
			resp.Diagnostics.AddError("Error listing product environments", err.Error())
			return
		}

		var found *productEnvironmentDataSourceModel
		if list != nil {
			for i := range list.SubAccounts {
				env := &list.SubAccounts[i]
				if util.DerefString(env.CloudName) == cloudName {
					m := productEnvironmentDataSourceModel{
						ID:        types.StringValue(util.DerefString(env.ID)),
						Name:      types.StringValue(util.DerefString(env.Name)),
						CloudName: types.StringValue(util.DerefString(env.CloudName)),
						Enabled:   types.BoolValue(util.DerefBool(env.Enabled)),
					}
					if env.CreatedAt != nil {
						m.CreatedAt = types.StringValue(env.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
					} else {
						m.CreatedAt = types.StringValue("")
					}
					if env.UpdatedAt != nil {
						m.UpdatedAt = types.StringValue(env.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
					} else {
						m.UpdatedAt = types.StringValue("")
					}
					found = &m
					break
				}
			}
		}

		if found == nil {
			resp.Diagnostics.AddError(
				"Product environment not found",
				fmt.Sprintf("No product environment with cloud_name %q was found.", cloudName),
			)
			return
		}
		data = *found
	} else {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Either id or cloud_name must be set to look up a product environment.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
