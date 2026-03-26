package provider

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewRoleDataSource(t *testing.T) {
	t.Parallel()

	ds := NewRoleDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*RoleDataSource); !ok {
		t.Fatalf("expected *RoleDataSource, got %T", ds)
	}
}

func TestRoleDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &RoleDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "nomatron"}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_role" {
		t.Fatalf("expected type name %q, got %q", "nomatron_role", resp.TypeName)
	}
}

func TestRoleDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &RoleDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "name", true, false, false)
	assertDataSourceStringAttribute(t, attrs, "description", false, false, true)
	assertDataSourceListAttribute(t, attrs, "permissions", false, false, true)
	assertDataSourceBoolAttribute(t, attrs, "built_in", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "scope", false, false, true)
}

func TestRoleDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &RoleDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, &resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestRoleDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &RoleDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "not-a-client"}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestRoleDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &RoleDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}

func TestFlattenRoleDataSource(t *testing.T) {
	t.Parallel()

	roleID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	description := "Read-only role"
	role := sdk.RoleDetail{
		Id:          roleID,
		Name:        "viewer",
		Description: &description,
		Permissions: []string{"global.roles.read", "global.users.read"},
		BuiltIn:     false,
		Scope:       "global",
	}

	data, diags := flattenRoleDataSource(role)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.ID.ValueString() != roleID.String() {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "viewer" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", data.Description.ValueString())
	}
	if data.BuiltIn.ValueBool() {
		t.Fatal("expected built_in=false")
	}
	if data.Scope.ValueString() != "global" {
		t.Fatalf("unexpected scope: %q", data.Scope.ValueString())
	}

	var permissions []string
	diags = data.Permissions.ElementsAs(context.Background(), &permissions, false)
	if diags.HasError() {
		t.Fatalf("unexpected permissions diagnostics: %v", diags)
	}
	if len(permissions) != 2 {
		t.Fatalf("unexpected permissions length: %#v", permissions)
	}
}

func TestFlattenRoleDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	roleID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	role := sdk.RoleDetail{
		Id:          roleID,
		Name:        "viewer",
		Permissions: []string{"global.roles.read"},
		BuiltIn:     true,
	}

	data, diags := flattenRoleDataSource(role)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !data.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !data.Scope.IsNull() {
		t.Fatal("expected scope to be null")
	}
	if !data.BuiltIn.ValueBool() {
		t.Fatal("expected built_in=true")
	}
}

func assertDataSourceListAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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
