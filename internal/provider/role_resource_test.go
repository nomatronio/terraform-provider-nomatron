package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewRoleResource(t *testing.T) {
	t.Parallel()

	r := NewRoleResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*RoleResource); !ok {
		t.Fatalf("expected *RoleResource, got %T", r)
	}
}

func TestRoleResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &RoleResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_role" {
		t.Fatalf("expected type name %q, got %q", "nomatron_role", resp.TypeName)
	}
}

func TestRoleResource_Schema(t *testing.T) {
	t.Parallel()

	r := &RoleResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, false, false)
	assertResourceListAttribute(t, attrs, "permissions", true, false, false)
	assertResourceBoolAttribute(t, attrs, "built_in", false, false, true)
	assertResourceStringAttribute(t, attrs, "scope", false, false, true, false)

	nameAttr := attrs["name"].(schema.StringAttribute)
	if len(nameAttr.PlanModifiers) != 1 {
		t.Fatalf("expected name to have 1 plan modifier, got %d", len(nameAttr.PlanModifiers))
	}
}

func TestRoleResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &RoleResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestRoleResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &RoleResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestRoleResource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	r := &RoleResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if r.client != client {
		t.Fatal("expected configured client to be stored on resource")
	}
}

func TestStateFromRole(t *testing.T) {
	t.Parallel()

	roleID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	description := "Read-only role"
	base := RoleResourceModel{
		Description: types.StringValue("old description"),
	}

	role := sdk.RoleDetail{
		Id:          roleID,
		Name:        "viewer",
		Description: &description,
		Permissions: []string{"global.roles.read", "global.users.read"},
		BuiltIn:     false,
		Scope:       "global",
	}

	state, diags := stateFromRole(base, role)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ID.ValueString() != roleID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "viewer" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if state.Scope.ValueString() != "global" {
		t.Fatalf("unexpected scope: %q", state.Scope.ValueString())
	}
	if state.BuiltIn.ValueBool() {
		t.Fatal("expected built_in=false")
	}

	var permissions []string
	diags = state.Permissions.ElementsAs(context.Background(), &permissions, false)
	if diags.HasError() {
		t.Fatalf("unexpected permissions diagnostics: %v", diags)
	}
	if len(permissions) != 2 || permissions[0] != "global.roles.read" || permissions[1] != "global.users.read" {
		t.Fatalf("unexpected permissions: %#v", permissions)
	}
}

func TestStateFromRole_PreservesOptionalDescriptionWhenMissing(t *testing.T) {
	t.Parallel()

	roleID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	basePermissions, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"global.roles.read"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building list: %v", diags)
	}

	base := RoleResourceModel{
		Description: types.StringValue("preserve me"),
		Permissions: basePermissions,
	}

	role := sdk.RoleDetail{
		Id:          roleID,
		Name:        "viewer",
		Description: nil,
		Permissions: []string{"global.roles.read"},
		BuiltIn:     true,
	}

	state, diags := stateFromRole(base, role)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.Description.ValueString() != "preserve me" {
		t.Fatalf("expected description to be preserved, got %q", state.Description.ValueString())
	}
	if !state.Scope.IsNull() {
		t.Fatalf("expected null scope, got %q", state.Scope.ValueString())
	}
	if !state.BuiltIn.ValueBool() {
		t.Fatal("expected built_in=true")
	}
}

func TestTerraformListToStringSlice(t *testing.T) {
	t.Parallel()

	in, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"a", "b"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building list: %v", diags)
	}

	out, diags := terraformListToStringSlice(context.Background(), in)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestTerraformListToStringSlice_Null(t *testing.T) {
	t.Parallel()

	out, diags := terraformListToStringSlice(context.Background(), types.ListNull(types.StringType))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if out != nil {
		t.Fatalf("expected nil output, got %#v", out)
	}
}

func TestRoleNotFoundError(t *testing.T) {
	t.Parallel()

	err := &roleNotFoundError{name: "viewer"}
	if !isRoleNotFound(err) {
		t.Fatal("expected roleNotFoundError to be recognized")
	}
	if isRoleNotFound(errors.New("other")) {
		t.Fatal("expected non-role error to be ignored")
	}
}

func assertResourceListAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	listAttr, ok := attr.(schema.ListAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.ListAttribute, got %T", name, attr)
	}

	if listAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, listAttr.Required)
	}
	if listAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, listAttr.Optional)
	}
	if listAttr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, listAttr.Computed)
	}
}
