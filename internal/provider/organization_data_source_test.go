package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewOrganizationDataSource(t *testing.T) {
	t.Parallel()

	ds := NewOrganizationDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*OrganizationDataSource); !ok {
		t.Fatalf("expected *OrganizationDataSource, got %T", ds)
	}
}

func TestOrganizationDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &OrganizationDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "nomatron"}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_organization" {
		t.Fatalf("expected type name %q, got %q", "nomatron_organization", resp.TypeName)
	}
}

func TestOrganizationDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &OrganizationDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertDataSourceStringAttribute(t, attrs, "id", false, true, true)
	assertDataSourceStringAttribute(t, attrs, "name", false, true, true)
	assertDataSourceStringAttribute(t, attrs, "description", false, false, true)
	assertDataSourceMapAttribute(t, attrs, "metadata", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "owner_user_id", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "created_at", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "updated_at", false, false, true)
}

func TestOrganizationDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &OrganizationDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, &resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestOrganizationDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &OrganizationDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "not-a-client"}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestOrganizationDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &OrganizationDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}

func TestFlattenOrganizationDataSource(t *testing.T) {
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

	org := sdk.Organization{
		Id:          orgID,
		Name:        "platform",
		Description: &description,
		Metadata:    &metadata,
		OwnerUserId: ownerUserID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	data, diags := flattenOrganizationDataSource(org)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.ID.ValueString() != orgID.String() {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "platform" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", data.Description.ValueString())
	}
	if data.OwnerUserID.ValueString() != ownerUserID.String() {
		t.Fatalf("unexpected owner_user_id: %q", data.OwnerUserID.ValueString())
	}
	if data.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", data.CreatedAt.ValueString())
	}
	if data.UpdatedAt.ValueString() != updatedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", data.UpdatedAt.ValueString())
	}
}

func TestFlattenOrganizationDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	orgID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	ownerUserID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))

	org := sdk.Organization{
		Id:          orgID,
		Name:        "platform",
		OwnerUserId: ownerUserID,
	}

	data, diags := flattenOrganizationDataSource(org)
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
