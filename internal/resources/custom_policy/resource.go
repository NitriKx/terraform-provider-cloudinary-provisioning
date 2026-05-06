package custom_policy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudinary/account-provisioning-go/cldprovisioning"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/components"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/operations"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/sdkerrors"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/optionalnullable"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &customPolicyResource{}
var _ resource.ResourceWithImportState = &customPolicyResource{}
var _ resource.ResourceWithConfigValidators = &customPolicyResource{}

// NewResource returns a new cloudinaryprovisioning_custom_policy resource.
func NewResource() resource.Resource {
	return &customPolicyResource{}
}

type customPolicyResource struct {
	client *cldprovisioning.CldProvisioning
}

type customPolicyModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	PolicyStatement types.String `tfsdk:"policy_statement"`
	Description     types.String `tfsdk:"description"`
	ScopeType       types.String `tfsdk:"scope_type"`
	ScopeID         types.String `tfsdk:"scope_id"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func (r *customPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_policy"
}

func (r *customPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudinary custom permissions policy. " +
			"Custom policies allow fine-grained access control over Cloudinary resources using Cedar policy language.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the custom policy assigned by Cloudinary.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The display name of the custom policy.",
			},
			"policy_statement": schema.StringAttribute{
				Required:    true,
				Description: "The Cedar policy statement string.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A human-readable description of the custom policy.",
			},
			"scope_type": schema.StringAttribute{
				Required: true,
				Description: "The scope of the policy. Must be one of: " +
					"\"account\" (applies to the whole account) or " +
					"\"prodenv\" (applies to a specific product environment).",
				Validators: []validator.String{
					stringvalidator.OneOf("account", "prodenv"),
				},
			},
			"scope_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The ID of the product environment this policy applies to. " +
					"Required when scope_type is \"prodenv\". " +
					"Must be omitted or empty when scope_type is \"account\".",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the custom policy is enabled.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The Unix timestamp at which the custom policy was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The Unix timestamp at which the custom policy was last updated.",
			},
		},
	}
}

// ConfigValidators enforces that scope_id is required when scope_type is "prodenv".
func (r *customPolicyResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		scopeIDValidator{},
	}
}

// scopeIDValidator validates the scope_id / scope_type relationship.
type scopeIDValidator struct{}

func (v scopeIDValidator) Description(_ context.Context) string {
	return "scope_id is required when scope_type is \"prodenv\""
}

func (v scopeIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v scopeIDValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var scopeType types.String
	var scopeID types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("scope_type"), &scopeType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("scope_id"), &scopeID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if scopeType.IsUnknown() || scopeType.IsNull() {
		return
	}

	if scopeType.ValueString() == "prodenv" {
		if scopeID.IsUnknown() {
			return
		}
		if scopeID.IsNull() || scopeID.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("scope_id"),
				"scope_id required",
				"scope_id must be set when scope_type is \"prodenv\".",
			)
		}
	}
}

func (r *customPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *customPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := components.CustomPolicy{
		Name:            plan.Name.ValueString(),
		PolicyStatement: plan.PolicyStatement.ValueString(),
		ScopeType:       components.ScopeTypeEnum(plan.ScopeType.ValueString()),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		body.Description = optionalnullable.From(&desc)
	}
	if !plan.ScopeID.IsNull() && !plan.ScopeID.IsUnknown() {
		sid := plan.ScopeID.ValueString()
		body.ScopeID = optionalnullable.From(&sid)
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled := plan.Enabled.ValueBool()
		body.Enabled = optionalnullable.From(&enabled)
	}

	result, err := r.client.CustomPolicies.Create(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom policy", err.Error())
		return
	}

	// Preserve the plan's policy_statement verbatim — the API normalises whitespace,
	// which would cause an "inconsistent result after apply" error if we stored the
	// API-returned value instead.
	state := policyFromResult(result)
	state.PolicyStatement = plan.PolicyStatement
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CustomPolicies.Get(ctx, state.ID.ValueString())
	if err != nil {
		var errResp *sdkerrors.ErrorResponse
		if errors.As(err, &errResp) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading custom policy", err.Error())
		return
	}
	if result == nil || result.Data == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	newState := policyFromResult(result)
	// If the API's normalised policy matches the state's version (only whitespace differs),
	// preserve the user's original formatting to avoid spurious diffs on the next plan.
	if normalizePolicyStatement(newState.PolicyStatement.ValueString()) ==
		normalizePolicyStatement(state.PolicyStatement.ValueString()) {
		newState.PolicyStatement = state.PolicyStatement
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *customPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state customPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateBody := components.UpdateCustomPolicy{
		Name:            plan.Name.ValueString(),
		PolicyStatement: plan.PolicyStatement.ValueString(),
		ScopeType:       components.ScopeTypeEnum(plan.ScopeType.ValueString()),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		updateBody.Description = optionalnullable.From(&desc)
	}
	if !plan.ScopeID.IsNull() && !plan.ScopeID.IsUnknown() {
		sid := plan.ScopeID.ValueString()
		updateBody.ScopeID = optionalnullable.From(&sid)
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled := plan.Enabled.ValueBool()
		updateBody.Enabled = optionalnullable.From(&enabled)
	}

	result, err := r.client.CustomPolicies.Update(ctx, operations.UpdateCustomPolicyRequest{
		PolicyID:           state.ID.ValueString(),
		UpdateCustomPolicy: updateBody,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom policy", err.Error())
		return
	}

	// Preserve the plan's policy_statement verbatim (same reason as Create).
	newState := policyFromResult(result)
	newState.PolicyStatement = plan.PolicyStatement
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *customPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.CustomPolicies.Delete(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting custom policy", err.Error())
		return
	}
}

func (r *customPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &customPolicyModel{
		ID: types.StringValue(req.ID),
	})...)
}

// normalizePolicyStatement collapses all whitespace sequences (including newlines and
// indentation) to a single space and trims surrounding whitespace.
func normalizePolicyStatement(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// policyFromResult maps a Cloudinary API result to the Terraform state model.
func policyFromResult(result *components.CustomPolicyResponse) customPolicyModel {
	if result == nil || result.Data == nil {
		return customPolicyModel{}
	}
	d := result.Data

	description, _ := d.Description.GetOrZero()
	scopeID, _ := d.ScopeID.GetOrZero()

	return customPolicyModel{
		ID:              types.StringValue(d.ID),
		Name:            types.StringValue(d.Name),
		PolicyStatement: types.StringValue(d.PolicyStatement),
		Description:     types.StringValue(description),
		ScopeType:       types.StringValue(string(d.ScopeType)),
		ScopeID:         types.StringValue(scopeID),
		Enabled:         types.BoolValue(d.Enabled),
		CreatedAt:       types.StringValue(time.Unix(d.CreatedAt, 0).UTC().Format("2006-01-02T15:04:05Z")),
		UpdatedAt:       types.StringValue(time.Unix(d.UpdatedAt, 0).UTC().Format("2006-01-02T15:04:05Z")),
	}
}
