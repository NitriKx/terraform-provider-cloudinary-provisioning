package access_key

import (
	"context"
	"testing"
	"time"

	"github.com/cloudinary/account-provisioning-go/cldprovisioning"
	"github.com/cloudinary/account-provisioning-go/cldprovisioning/models/components"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Every other resource in this provider exposes a computed "id". access_key did
// not, which left consumers that read the Terraform state ID with nothing to key
// on: they fail with "cannot find id in tfstate" even though the key was created.
func TestSchemaExposesComputedID(t *testing.T) {
	r := &accessKeyResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal(`schema is missing the "id" attribute`)
	}
	if !attr.IsComputed() {
		t.Error(`"id" must be Computed: Cloudinary assigns the api_key server-side`)
	}
}

// The ID format must match what ImportState already parses, so that an imported
// resource and a freshly created one carry the same identifier.
func TestAccessKeyFromResultSetsCompositeID(t *testing.T) {
	createdAt := time.Date(2026, 7, 28, 13, 35, 34, 0, time.UTC)
	result := &components.AccessKey{
		Name:      cldprovisioning.String("kuttl-test-preprod-eu"),
		Enabled:   cldprovisioning.Bool(true),
		APIKey:    cldprovisioning.String("123456789012345"),
		APISecret: cldprovisioning.String("s3cret"),
		CreatedAt: &createdAt,
	}

	state := accessKeyFromResult(result, types.StringValue("75710c684b84ea111325a6d58c43cd"))

	want := "75710c684b84ea111325a6d58c43cd/123456789012345"
	if got := state.ID.ValueString(); got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
}

// ImportState receives the composite ID directly; it must survive into state
// rather than being left null, otherwise the first refresh after an import
// reports a missing identifier.
func TestAccessKeyIDRoundTripsThroughImportFormat(t *testing.T) {
	const productEnvironmentID = "75710c684b84ea111325a6d58c43cd"
	const apiKey = "123456789012345"

	id := accessKeyID(productEnvironmentID, apiKey)

	gotEnv, gotKey, err := parseAccessKeyID(id)
	if err != nil {
		t.Fatalf("parseAccessKeyID(%q) returned error: %v", id, err)
	}
	if gotEnv != productEnvironmentID {
		t.Errorf("product environment = %q, want %q", gotEnv, productEnvironmentID)
	}
	if gotKey != apiKey {
		t.Errorf("api key = %q, want %q", gotKey, apiKey)
	}
}

// An imported resource must land in state with its id already populated. If import
// only sets product_environment_id and api_key, the subsequent refresh preserves a
// null id and the resource is left without an identifier indefinitely.
func TestImportStatePopulatesID(t *testing.T) {
	r := &accessKeyResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	const importID = "75710c684b84ea111325a6d58c43cd/123456789012345"
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.ImportState(context.Background(), resource.ImportStateRequest{ID: importID}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState reported errors: %v", resp.Diagnostics.Errors())
	}

	var state accessKeyModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("reading back state: %v", diags.Errors())
	}
	if got := state.ID.ValueString(); got != importID {
		t.Errorf("ID = %q, want %q", got, importID)
	}
}

func TestImportStateRejectsMalformedID(t *testing.T) {
	r := &accessKeyResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "no-separator"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("expected a diagnostic error for a malformed import ID, got none")
	}
}

// Resources that were created before the id attribute existed carry state with a
// null id. A refresh has to backfill it, otherwise merely upgrading the provider
// leaves them without an identifier indefinitely.
func TestApplyFoundAccessKeyBackfillsMissingID(t *testing.T) {
	state := accessKeyModel{
		ID:                   types.StringNull(),
		ProductEnvironmentID: types.StringValue("75710c684b84ea111325a6d58c43cd"),
		APISecret:            types.StringValue("preserved-secret"),
	}
	found := &components.AccessKey{
		Name:    cldprovisioning.String("kuttl-test-preprod-eu"),
		Enabled: cldprovisioning.Bool(true),
		APIKey:  cldprovisioning.String("123456789012345"),
	}

	applyFoundAccessKey(&state, found)

	want := "75710c684b84ea111325a6d58c43cd/123456789012345"
	if got := state.ID.ValueString(); got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	// api_secret is never returned by the list endpoint, so it must survive a refresh.
	if got := state.APISecret.ValueString(); got != "preserved-secret" {
		t.Errorf("api_secret = %q, want %q", got, "preserved-secret")
	}
}

func TestParseAccessKeyIDRejectsMalformedInput(t *testing.T) {
	for _, id := range []string{"", "no-separator", "/missing-env", "missing-key/"} {
		if _, _, err := parseAccessKeyID(id); err == nil {
			t.Errorf("parseAccessKeyID(%q): expected an error, got nil", id)
		}
	}
}
