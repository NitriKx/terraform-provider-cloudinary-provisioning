package product_environment

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudinary/account-provisioning-go/cldprovisioning"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/components"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/operations"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/sdkerrors"

	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/providerdata"
	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/util"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &productEnvironmentResource{}
var _ resource.ResourceWithImportState = &productEnvironmentResource{}
var _ resource.ResourceWithModifyPlan = &productEnvironmentResource{}

// NewResource returns a new cloudinaryprovisioning_product_environment resource.
func NewResource() resource.Resource {
	return &productEnvironmentResource{}
}

type productEnvironmentResource struct {
	client *cldprovisioning.CldProvisioning
}

type productEnvironmentModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	CloudName          types.String `tfsdk:"cloud_name"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	BaseSubAccountID   types.String `tfsdk:"base_sub_account_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	DeletionProtection types.Bool   `tfsdk:"deletion_protection"`
}

func (r *productEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_environment"
}

func (r *productEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudinary product environment (sub-account). " +
			"Product environments allow you to segment your Cloudinary usage for different projects or teams.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the product environment assigned by Cloudinary.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The display name of the product environment.",
			},
			"cloud_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Cloudinary cloud name for this product environment. Auto-generated if not provided.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the product environment is enabled.",
			},
			"base_sub_account_id": schema.StringAttribute{
				Optional: true,
				Description: "The ID of an existing product environment to copy settings from when creating this one. " +
					"Changing this value forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the product environment was created (RFC3339).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the product environment was last updated (RFC3339).",
			},
			"deletion_protection": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Whether to prevent accidental deletion of this product environment. " +
					"Defaults to true. When true, terraform destroy will fail with an error. " +
					"To destroy the resource, set this to false and run terraform apply first.",
			},
		},
	}
}

func (r *productEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = pd.Client
}

func (r *productEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan productEnvironmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := components.ProductEnvironmentRequest{
		Name: plan.Name.ValueString(),
	}
	if !plan.CloudName.IsNull() && !plan.CloudName.IsUnknown() {
		body.CloudName = cldprovisioning.String(plan.CloudName.ValueString())
	}
	if !plan.BaseSubAccountID.IsNull() && !plan.BaseSubAccountID.IsUnknown() {
		body.BaseSubAccountID = cldprovisioning.String(plan.BaseSubAccountID.ValueString())
	}

	result, err := r.client.ProductEnvironments.Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating product environment", err.Error())
		return
	}

	state := productEnvironmentFromResult(result, plan.BaseSubAccountID, plan.DeletionProtection)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *productEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state productEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ProductEnvironments.Get(ctx, state.ID.ValueString())
	if err != nil {
		var permErr *sdkerrors.PermissionsErrorResponse
		if errors.As(err, &permErr) {
			// 403: caller lacks read permission on this product environment.
			// Keep existing state so that Terraform can still plan and execute Delete().
			return
		}
		var errResp *sdkerrors.ErrorResponse
		if errors.As(err, &errResp) {
			// 404 or other API-level error: resource was deleted outside Terraform.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading product environment", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	deletionProtection := state.DeletionProtection
	if deletionProtection.IsNull() {
		deletionProtection = types.BoolValue(true)
	}
	newState := productEnvironmentFromResult(result, state.BaseSubAccountID, deletionProtection)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *productEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan productEnvironmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state productEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateBody := components.ProductEnvironmentUpdateRequest{
		Name: cldprovisioning.String(plan.Name.ValueString()),
	}
	if !plan.CloudName.IsNull() && !plan.CloudName.IsUnknown() {
		updateBody.CloudName = cldprovisioning.String(plan.CloudName.ValueString())
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		updateBody.Enabled = cldprovisioning.Bool(plan.Enabled.ValueBool())
	}

	result, err := r.client.ProductEnvironments.Update(ctx, operations.UpdateProductEnvironmentRequest{
		SubAccountID:                    state.ID.ValueString(),
		ProductEnvironmentUpdateRequest: updateBody,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating product environment", err.Error())
		return
	}

	newState := productEnvironmentFromResult(result, plan.BaseSubAccountID, plan.DeletionProtection)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *productEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state productEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion protection enabled",
			fmt.Sprintf(
				"Resource cloudinaryprovisioning_product_environment %q has deletion_protection = true. "+
					"Set deletion_protection = false and run terraform apply before destroying.",
				state.Name.ValueString(),
			),
		)
		return
	}

	_, err := r.client.ProductEnvironments.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting product environment", err.Error())
		return
	}
}

// ModifyPlan blocks destruction at plan time when deletion_protection is enabled,
// preventing dependent resources from being partially destroyed before the error surfaces.
func (r *productEnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if !req.Plan.Raw.IsNull() {
		return
	}
	var state productEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Deletion protection enabled",
			fmt.Sprintf(
				"Resource cloudinaryprovisioning_product_environment %q has deletion_protection = true. "+
					"Set deletion_protection = false and run terraform apply before destroying.",
				state.Name.ValueString(),
			),
		)
	}
}

func (r *productEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &productEnvironmentModel{
		ID:                 types.StringValue(req.ID),
		DeletionProtection: types.BoolValue(true),
	})...)
}

// productEnvironmentFromResult maps a Cloudinary API result to the Terraform state model.
func productEnvironmentFromResult(result *components.ProductEnvironment, baseSubAccountID types.String, deletionProtection types.Bool) productEnvironmentModel {
	state := productEnvironmentModel{
		ID:                 types.StringValue(util.DerefString(result.ID)),
		Name:               types.StringValue(util.DerefString(result.Name)),
		CloudName:          types.StringValue(util.DerefString(result.CloudName)),
		Enabled:            types.BoolValue(util.DerefBool(result.Enabled)),
		BaseSubAccountID:   baseSubAccountID,
		DeletionProtection: deletionProtection,
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
