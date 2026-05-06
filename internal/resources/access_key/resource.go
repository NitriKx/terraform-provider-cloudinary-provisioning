package access_key

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudinary/account-provisioning-go/cldprovisioning"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/components"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/operations"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/sdkerrors"

	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &accessKeyResource{}
var _ resource.ResourceWithImportState = &accessKeyResource{}

// NewResource returns a new cloudinaryprovisioning_access_key resource.
func NewResource() resource.Resource {
	return &accessKeyResource{}
}

type accessKeyResource struct {
	client *cldprovisioning.CldProvisioning
}

type accessKeyModel struct {
	ProductEnvironmentID types.String `tfsdk:"product_environment_id"`
	Name                 types.String `tfsdk:"name"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	APIKey               types.String `tfsdk:"api_key"`
	APISecret            types.String `tfsdk:"api_secret"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (r *accessKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_key"
}

func (r *accessKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API access key for a Cloudinary product environment. " +
			"Access keys provide the credentials needed to interact with a product environment's API.",
		Attributes: map[string]schema.Attribute{
			"product_environment_id": schema.StringAttribute{
				Required: true,
				Description: "The ID of the product environment this access key belongs to. " +
					"Changing this value forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A display name for the access key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the access key is enabled.",
			},
			"api_key": schema.StringAttribute{
				Computed:    true,
				Description: "The API key identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The API secret. Only available immediately after creation. Preserved in state.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the access key was created (RFC3339).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the access key was last updated (RFC3339).",
			},
		},
	}
}

func (r *accessKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*cldprovisioning.CldProvisioning)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *cldprovisioning.CldProvisioning, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *accessKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := components.AccessKeyRequest{}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		body.Name = cldprovisioning.String(plan.Name.ValueString())
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		body.Enabled = cldprovisioning.Bool(plan.Enabled.ValueBool())
	}

	result, err := r.client.AccessKeys.Generate(ctx, operations.GenerateAccessKeyRequest{
		SubAccountID:     plan.ProductEnvironmentID.ValueString(),
		AccessKeyRequest: body,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating access key", err.Error())
		return
	}

	state := accessKeyFromResult(result, plan.ProductEnvironmentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accessKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := findAccessKey(ctx, r.client, state.ProductEnvironmentID.ValueString(), state.APIKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading access key", err.Error())
		return
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// api_secret is not returned on reads; preserve the value from state.
	state.Name = types.StringValue(util.DerefString(found.Name))
	state.Enabled = types.BoolValue(util.DerefBool(found.Enabled))
	state.APIKey = types.StringValue(util.DerefString(found.APIKey))
	if found.CreatedAt != nil {
		state.CreatedAt = types.StringValue(found.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
	}
	if found.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(found.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
	}
	// api_secret stays as-is from state.

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accessKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan accessKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state accessKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateBody := components.AccessKeyUpdateRequest{}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		updateBody.Name = cldprovisioning.String(plan.Name.ValueString())
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		updateBody.Enabled = cldprovisioning.Bool(plan.Enabled.ValueBool())
	}

	result, err := r.client.AccessKeys.Update(ctx, operations.UpdateAccessKeyRequest{
		SubAccountID:           state.ProductEnvironmentID.ValueString(),
		Key:                    state.APIKey.ValueString(),
		AccessKeyUpdateRequest: updateBody,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating access key", err.Error())
		return
	}

	// Preserve api_secret from state since it's not returned on updates.
	state.Name = types.StringValue(util.DerefString(result.Name))
	state.Enabled = types.BoolValue(util.DerefBool(result.Enabled))
	state.APIKey = types.StringValue(util.DerefString(result.APIKey))
	if result.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(result.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accessKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.AccessKeys.Delete(ctx, operations.DeleteAccessKeyRequest{
		SubAccountID: state.ProductEnvironmentID.ValueString(),
		Key:          state.APIKey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting access key", err.Error())
		return
	}
}

func (r *accessKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "{product_environment_id}/{api_key}"
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf(
				"Expected format: {product_environment_id}/{api_key}. Got: %q",
				req.ID,
			),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &accessKeyModel{
		ProductEnvironmentID: types.StringValue(parts[0]),
		APIKey:               types.StringValue(parts[1]),
	})...)
}

// findAccessKey looks up an access key by its API key value within a product environment,
// paginating through all pages until it is found or all results are exhausted.
// Returns nil, nil if not found.
func findAccessKey(ctx context.Context, client *cldprovisioning.CldProvisioning, subAccountID, apiKey string) (*components.AccessKey, error) {
	pageSize := int64(100)
	page := int64(1)

	for {
		result, err := client.AccessKeys.List(ctx, operations.GetAccessKeysRequest{
			SubAccountID: subAccountID,
			PageSize:     &pageSize,
			Page:         &page,
		})
		if err != nil {
			var errResp *sdkerrors.ErrorResponse
			if errors.As(err, &errResp) {
				return nil, nil
			}
			return nil, err
		}
		if result == nil || len(result.AccessKeys) == 0 {
			return nil, nil
		}

		for i := range result.AccessKeys {
			k := &result.AccessKeys[i]
			if util.DerefString(k.APIKey) == apiKey {
				return k, nil
			}
		}

		// If the page returned fewer results than the page size, we've reached the last page.
		if int64(len(result.AccessKeys)) < pageSize {
			return nil, nil
		}
		page++
	}
}

// accessKeyFromResult maps a Cloudinary API result to the Terraform state model.
func accessKeyFromResult(result *components.AccessKey, productEnvironmentID types.String) accessKeyModel {
	state := accessKeyModel{
		ProductEnvironmentID: productEnvironmentID,
		Name:                 types.StringValue(util.DerefString(result.Name)),
		Enabled:              types.BoolValue(util.DerefBool(result.Enabled)),
		APIKey:               types.StringValue(util.DerefString(result.APIKey)),
		APISecret:            types.StringValue(util.DerefString(result.APISecret)),
	}
	if result.CreatedAt != nil {
		state.CreatedAt = types.StringValue(result.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
	} else {
		state.CreatedAt = types.StringValue("")
	}
	if result.UpdatedAt != nil {
		state.UpdatedAt = types.StringValue(result.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
	} else {
		state.UpdatedAt = types.StringValue("")
	}
	return state
}
