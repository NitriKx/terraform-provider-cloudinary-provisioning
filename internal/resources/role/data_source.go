package role

import (
	"context"
	"fmt"

	"github.com/cloudinary/account-provisioning-go/cldprovisioning"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/components"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/operations"

	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/providerdata"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &roleDataSource{}

// NewDataSource returns a new cloudinaryprovisioning_role data source.
func NewDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	client *cldprovisioning.CldProvisioning
}

type roleDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	PermissionType types.String `tfsdk:"permission_type"`
	Description    types.String `tfsdk:"description"`
	ManagementType types.String `tfsdk:"management_type"`
	ScopeType      types.String `tfsdk:"scope_type"`
}

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Cloudinary role by name and permission type. " +
			"Use this data source to retrieve the role ID needed for principal role assignments.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the role.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the role to look up (e.g. \"Admin\").",
			},
			"permission_type": schema.StringAttribute{
				Required: true,
				Description: "The permission type of the role. Must be one of: " +
					"\"global\" (role applies across all contexts within its scope, e.g. full account or all folders in a product environment) or " +
					"\"content\" (role applies to specific content instances such as a particular folder or collection).",
				Validators: []validator.String{
					stringvalidator.OneOf("global", "content"),
				},
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "A short description of the role.",
			},
			"management_type": schema.StringAttribute{
				Computed:    true,
				Description: "Whether the role is managed by Cloudinary (\"system\") or by the user (\"custom\").",
			},
			"scope_type": schema.StringAttribute{
				Computed:    true,
				Description: "Where the role is applied: \"account\" or \"prodenv\".",
			},
		},
	}
}

func (d *roleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data roleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permType := components.PermissionTypeEnum(data.PermissionType.ValueString())
	result, err := d.client.Roles.List(ctx, operations.GetRolesRequest{
		PermissionType: permType,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error listing roles", err.Error())
		return
	}
	if result == nil {
		resp.Diagnostics.AddError("Error listing roles", "API returned empty response")
		return
	}

	name := data.Name.ValueString()
	for _, r := range result.Data {
		if r.Name == name {
			description, _ := r.Description.GetOrZero()
			data.ID = types.StringValue(r.ID)
			data.Description = types.StringValue(description)
			data.ManagementType = types.StringValue(string(r.ManagementType))
			data.ScopeType = types.StringValue(string(r.ScopeType))
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Role not found",
		fmt.Sprintf("No role named %q with permission_type %q was found.", name, permType),
	)
}
