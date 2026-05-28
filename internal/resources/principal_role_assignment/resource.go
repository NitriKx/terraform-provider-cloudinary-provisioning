package principal_role_assignment

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudinary/account-provisioning-go/cldprovisioning"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/components"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/operations"

	"github.com/NitriKx/terraform-provider-cloudinary-provisioning/internal/providerdata"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &principalRoleAssignmentResource{}
var _ resource.ResourceWithImportState = &principalRoleAssignmentResource{}
var _ resource.ResourceWithConfigValidators = &principalRoleAssignmentResource{}

// NewResource returns a new cloudinaryprovisioning_principal_role_assignment resource.
func NewResource() resource.Resource {
	return &principalRoleAssignmentResource{}
}

type principalRoleAssignmentResource struct {
	client *cldprovisioning.CldProvisioning
}

type principalRoleAssignmentModel struct {
	ID            types.String `tfsdk:"id"`
	PrincipalType types.String `tfsdk:"principal_type"`
	PrincipalID   types.String `tfsdk:"principal_id"`
	RoleID        types.String `tfsdk:"role_id"`
	ScopeType     types.String `tfsdk:"scope_type"`
	ScopeID       types.String `tfsdk:"scope_id"`
}

func (r *principalRoleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_principal_role_assignment"
}

func (r *principalRoleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a role to a principal (user, group, API key, or provisioning key) " +
			"within a given scope. All attributes are immutable: any change forces resource replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier: {principal_type}/{principal_id}/{role_id}/{scope_type}/{scope_id}.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"principal_type": schema.StringAttribute{
				Required: true,
				Description: "The type of principal. Must be one of: " +
					"\"user\", \"group\", \"apiKey\", \"provisioningKey\".",
				Validators: []validator.String{
					stringvalidator.OneOf("user", "group", "apiKey", "provisioningKey"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the principal.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the role to assign.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_type": schema.StringAttribute{
				Required: true,
				Description: "The scope level. Must be one of: " +
					"\"account\" (whole account) or \"prodenv\" (specific product environment).",
				Validators: []validator.String{
					stringvalidator.OneOf("account", "prodenv"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"scope_id": schema.StringAttribute{
				Optional: true,
				Description: "The ID of the product environment. " +
					"Required when scope_type is \"prodenv\". Must be omitted when scope_type is \"account\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// ConfigValidators enforces that scope_id is required when scope_type is "prodenv".
func (r *principalRoleAssignmentResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		scopeIDValidator{},
	}
}

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

func (r *principalRoleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *principalRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan principalRoleAssignmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	op := components.OperationEnumAdd
	body := components.UpdatePrincipalRolesRequest{
		Operation: &op,
		Principal: &components.Principal{
			Type: components.PrincipalTypeEnum(plan.PrincipalType.ValueString()),
			ID:   plan.PrincipalID.ValueString(),
		},
		Roles: []components.RoleToManage{
			buildRoleToManage(plan.RoleID.ValueString(), plan.ScopeID),
		},
	}

	if err := r.client.Principals.UpdateRoles(ctx, body); err != nil {
		resp.Diagnostics.AddError("Error assigning role", err.Error())
		return
	}

	plan.ID = types.StringValue(compositeID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *principalRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state principalRoleAssignmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.findAssignment(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Error reading role assignment", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *principalRoleAssignmentResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes are ForceNew — Update is never called.
}

func (r *principalRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state principalRoleAssignmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	op := components.OperationEnumRemove
	body := components.UpdatePrincipalRolesRequest{
		Operation: &op,
		Principal: &components.Principal{
			Type: components.PrincipalTypeEnum(state.PrincipalType.ValueString()),
			ID:   state.PrincipalID.ValueString(),
		},
		Roles: []components.RoleToManage{
			buildRoleToManage(state.RoleID.ValueString(), state.ScopeID),
		},
	}

	if err := r.client.Principals.UpdateRoles(ctx, body); err != nil {
		resp.Diagnostics.AddError("Error removing role assignment", err.Error())
		return
	}
}

func (r *principalRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "{principal_type}/{principal_id}/{role_id}/{scope_type}/{scope_id}"
	// scope_id may be empty when scope_type is "account".
	parts := strings.SplitN(req.ID, "/", 5)
	if len(parts) != 5 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf(
				"Expected format: {principal_type}/{principal_id}/{role_id}/{scope_type}/{scope_id}. Got: %q",
				req.ID,
			),
		)
		return
	}

	m := principalRoleAssignmentModel{
		PrincipalType: types.StringValue(parts[0]),
		PrincipalID:   types.StringValue(parts[1]),
		RoleID:        types.StringValue(parts[2]),
		ScopeType:     types.StringValue(parts[3]),
	}
	if parts[4] != "" {
		m.ScopeID = types.StringValue(parts[4])
	}
	m.ID = types.StringValue(compositeID(m))

	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

// findAssignment checks whether the role assignment still exists by querying the principal's roles.
// It tries both permission types (global, then content) since the permission type is not stored in state.
func (r *principalRoleAssignmentResource) findAssignment(ctx context.Context, state principalRoleAssignmentModel) (bool, error) {
	for _, permType := range []components.PermissionTypeEnum{
		components.PermissionTypeEnumGlobal,
		components.PermissionTypeEnumContent,
	} {
		scopeType := components.ScopeTypeEnum(state.ScopeType.ValueString())
		listReq := operations.GetPrincipalRolesRequest{
			PrincipalType:  components.PrincipalTypeEnum(state.PrincipalType.ValueString()),
			PrincipalID:    state.PrincipalID.ValueString(),
			PermissionType: permType,
			ScopeType:      &scopeType,
		}
		if !state.ScopeID.IsNull() && !state.ScopeID.IsUnknown() && state.ScopeID.ValueString() != "" {
			sid := state.ScopeID.ValueString()
			listReq.ScopeID = &sid
		}

		result, err := r.client.Principals.ListRoles(ctx, listReq)
		if err != nil {
			return false, err
		}
		if result == nil {
			continue
		}

		roleID := state.RoleID.ValueString()
		scopeID := ""
		if !state.ScopeID.IsNull() && !state.ScopeID.IsUnknown() {
			scopeID = state.ScopeID.ValueString()
		}

		for _, role := range result.Data {
			if role.ID != roleID {
				continue
			}
			assignedScopeID, _ := role.ScopeID.GetOrZero()
			if assignedScopeID == scopeID {
				return true, nil
			}
		}
	}
	return false, nil
}

// compositeID builds the resource ID from the model fields.
func compositeID(m principalRoleAssignmentModel) string {
	scopeID := ""
	if !m.ScopeID.IsNull() && !m.ScopeID.IsUnknown() {
		scopeID = m.ScopeID.ValueString()
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		m.PrincipalType.ValueString(),
		m.PrincipalID.ValueString(),
		m.RoleID.ValueString(),
		m.ScopeType.ValueString(),
		scopeID,
	)
}

// buildRoleToManage constructs the RoleToManage struct, setting ScopeID only when it is not empty.
func buildRoleToManage(roleID string, scopeID types.String) components.RoleToManage {
	r := components.RoleToManage{ID: roleID}
	if !scopeID.IsNull() && !scopeID.IsUnknown() && scopeID.ValueString() != "" {
		sid := scopeID.ValueString()
		r.ScopeID = &sid
	}
	return r
}
