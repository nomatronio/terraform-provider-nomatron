package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewOrganizationResource(t *testing.T) {
	t.Parallel()

	r := NewOrganizationResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*OrganizationResource); !ok {
		t.Fatalf("expected *OrganizationResource, got %T", r)
	}
}

func TestOrganizationResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &OrganizationResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_organization" {
		t.Fatalf("expected type name %q, got %q", "nomatron_organization", resp.TypeName)
	}
}

func TestOrganizationResource_Schema(t *testing.T) {
	t.Parallel()

	r := &OrganizationResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, false, false)
	assertResourceMapAttribute(t, attrs, "metadata", false, true, false)
	assertResourceStringAttribute(t, attrs, "owner_username", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "owner_user_id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "updated_at", false, false, true, false)
}

func TestOrganizationResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &OrganizationResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestOrganizationResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &OrganizationResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestStateFromOrganization(t *testing.T) {
	t.Parallel()

	orgID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	ownerUserID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	description := "Platform organization"
	metadata := map[string]interface{}{
		"team":  "platform",
		"owner": "terraform",
	}

	base := OrganizationResourceModel{
		OwnerUsername: types.StringValue("rbarnes"),
	}

	org := sdk.Organization{
		Id:          orgID,
		Name:        "platform",
		Description: &description,
		Metadata:    &metadata,
		OwnerUserId: ownerUserID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	state, diags := stateFromOrganization(base, org)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ID.ValueString() != orgID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "platform" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if state.OwnerUsername.ValueString() != "rbarnes" {
		t.Fatalf("unexpected owner_username: %q", state.OwnerUsername.ValueString())
	}
	if state.OwnerUserID.ValueString() != ownerUserID.String() {
		t.Fatalf("unexpected owner_user_id: %q", state.OwnerUserID.ValueString())
	}
	if state.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", state.CreatedAt.ValueString())
	}
	if state.UpdatedAt.ValueString() != updatedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", state.UpdatedAt.ValueString())
	}
}

func TestStateFromOrganization_PreservesOptionalFieldsWhenMissing(t *testing.T) {
	t.Parallel()

	base := OrganizationResourceModel{
		Description:   types.StringValue("preserve me"),
		Metadata:      types.MapValueMust(types.StringType, map[string]attr.Value{"team": types.StringValue("platform")}),
		OwnerUsername: types.StringValue("rbarnes"),
	}

	state, diags := stateFromOrganization(base, sdk.Organization{
		Id:          openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		Name:        "platform",
		OwnerUserId: openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")),
		CreatedAt:   time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.Description.ValueString() != "preserve me" {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if state.OwnerUsername.ValueString() != "rbarnes" {
		t.Fatalf("unexpected owner_username: %q", state.OwnerUsername.ValueString())
	}
}
