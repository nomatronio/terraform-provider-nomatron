package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewGroupDataSource(t *testing.T) {
	t.Parallel()

	ds := NewGroupDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*GroupDataSource); !ok {
		t.Fatalf("expected *GroupDataSource, got %T", ds)
	}
}

func TestGroupDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &GroupDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "nomatron"}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_group" {
		t.Fatalf("expected type name %q, got %q", "nomatron_group", resp.TypeName)
	}
}

func TestGroupDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &GroupDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertDataSourceStringAttribute(t, attrs, "name", true, false, false)
	assertDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "description", false, false, true)
	assertDataSourceMapAttribute(t, attrs, "metadata", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "organization_id", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "owner_user_id", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "created_at", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "updated_at", false, false, true)
}

func TestGroupDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &GroupDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, &resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestGroupDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &GroupDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "not-a-client"}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestGroupDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &GroupDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}

func TestFlattenGroupDataSource(t *testing.T) {
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

	base := GroupDataSourceModel{
		OrgName: types.StringValue("platform"),
		Name:    types.StringValue("admins"),
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

	data, diags := flattenGroupDataSource(base, group)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.ID.ValueString() != groupID.String() {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "admins" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", data.Description.ValueString())
	}
	if data.OrganizationID.ValueString() != orgID.String() {
		t.Fatalf("unexpected organization_id: %q", data.OrganizationID.ValueString())
	}
	if data.OwnerUserID.ValueString() != ownerID.String() {
		t.Fatalf("unexpected owner_user_id: %q", data.OwnerUserID.ValueString())
	}
}

func TestFlattenGroupDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	base := GroupDataSourceModel{
		OrgName: types.StringValue("platform"),
		Name:    types.StringValue("admins"),
	}

	group := sdk.Group{
		Id:             openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")),
		Name:           "admins",
		OrganizationId: openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")),
		OwnerUserId:    openapi_types.UUID(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")),
	}

	data, diags := flattenGroupDataSource(base, group)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !data.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !data.Metadata.IsNull() {
		t.Fatal("expected metadata to be null")
	}
	if !data.CreatedAt.IsNull() {
		t.Fatal("expected created_at to be null")
	}
	if !data.UpdatedAt.IsNull() {
		t.Fatal("expected updated_at to be null")
	}
}
