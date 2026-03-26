package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewGroupResource(t *testing.T) {
	t.Parallel()

	r := NewGroupResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*GroupResource); !ok {
		t.Fatalf("expected *GroupResource, got %T", r)
	}
}

func TestGroupResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &GroupResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_group" {
		t.Fatalf("expected type name %q, got %q", "nomatron_group", resp.TypeName)
	}
}

func TestGroupResource_Schema(t *testing.T) {
	t.Parallel()

	r := &GroupResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, false, false)
	assertResourceMapAttribute(t, attrs, "metadata", false, true, false)
	assertResourceStringAttribute(t, attrs, "owner_username", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "organization_id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "owner_user_id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "updated_at", false, false, true, false)
}

func TestGroupResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &GroupResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestGroupResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &GroupResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestStateFromGroup(t *testing.T) {
	t.Parallel()

	groupID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	orgID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	ownerID := openapi_types.UUID(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"))
	createdAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	description := "Platform admins"
	metadata := map[string]interface{}{
		"team": "platform",
	}

	base := GroupResourceModel{
		OrgName:       types.StringValue("platform"),
		OwnerUsername: types.StringValue("rbarnes"),
	}

	group := sdk.Group{
		Id:             groupID,
		Name:           "admins",
		Description:    &description,
		Metadata:       &metadata,
		OrganizationId: orgID,
		OwnerUserId:    ownerID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	state, diags := stateFromGroup(base, group)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ID.ValueString() != groupID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "admins" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if state.OrganizationID.ValueString() != orgID.String() {
		t.Fatalf("unexpected organization_id: %q", state.OrganizationID.ValueString())
	}
	if state.OwnerUserID.ValueString() != ownerID.String() {
		t.Fatalf("unexpected owner_user_id: %q", state.OwnerUserID.ValueString())
	}
}

func TestStateFromGroup_PreservesOptionalFieldsWhenMissing(t *testing.T) {
	t.Parallel()

	base := GroupResourceModel{
		OrgName:       types.StringValue("platform"),
		Description:   types.StringValue("preserve me"),
		Metadata:      types.MapValueMust(types.StringType, map[string]attr.Value{"team": types.StringValue("platform")}),
		OwnerUsername: types.StringValue("rbarnes"),
	}

	group := sdk.Group{
		Id:             openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		Name:           "admins",
		OrganizationId: openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")),
		OwnerUserId:    openapi_types.UUID(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")),
		CreatedAt:      time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
	}

	state, diags := stateFromGroup(base, group)
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

func TestGroupNotFoundError(t *testing.T) {
	t.Parallel()

	err := &groupNotFoundError{orgName: "platform", name: "admins"}
	if !isGroupNotFound(err) {
		t.Fatal("expected groupNotFoundError to be recognized")
	}
	if isGroupNotFound(errors.New("other")) {
		t.Fatal("expected non-group error to be ignored")
	}
}

func TestParseGroupImportID(t *testing.T) {
	t.Parallel()

	orgName, groupName, err := parseGroupImportID("platform/admins")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgName != "platform" || groupName != "admins" {
		t.Fatalf("unexpected parsed values: %q %q", orgName, groupName)
	}
}
