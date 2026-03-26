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

func TestNewServiceAccountDataSource(t *testing.T) {
	t.Parallel()

	ds := NewServiceAccountDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*ServiceAccountDataSource); !ok {
		t.Fatalf("expected *ServiceAccountDataSource, got %T", ds)
	}
}

func TestServiceAccountDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &ServiceAccountDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "nomatron"}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_service_account" {
		t.Fatalf("expected type name %q, got %q", "nomatron_service_account", resp.TypeName)
	}
}

func TestServiceAccountDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &ServiceAccountDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertDataSourceStringAttribute(t, attrs, "id", false, true, true)
	assertDataSourceStringAttribute(t, attrs, "name", false, true, true)
	assertDataSourceStringAttribute(t, attrs, "description", false, false, true)
	assertDataSourceBoolAttribute(t, attrs, "is_active", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "created_at", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "created_by", false, false, true)
}

func TestServiceAccountDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &ServiceAccountDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, &resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestServiceAccountDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &ServiceAccountDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "not-a-client"}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestServiceAccountDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &ServiceAccountDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}

func TestFlattenServiceAccountDataSource(t *testing.T) {
	t.Parallel()

	accountID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	creatorID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	description := "Automation service account"

	account := sdk.ServiceAccount{
		Id:          accountID,
		Name:        "terraform",
		Description: &description,
		IsActive:    true,
		CreatedAt:   createdAt,
		CreatedBy:   creatorID,
	}

	data := flattenServiceAccountDataSource(account)

	if data.ID.ValueString() != accountID.String() {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "terraform" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", data.Description.ValueString())
	}
	if !data.IsActive.ValueBool() {
		t.Fatal("expected is_active=true")
	}
	if data.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", data.CreatedAt.ValueString())
	}
	if data.CreatedBy.ValueString() != creatorID.String() {
		t.Fatalf("unexpected created_by: %q", data.CreatedBy.ValueString())
	}
}

func TestFlattenServiceAccountDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	accountID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	account := sdk.ServiceAccount{
		Id:       accountID,
		Name:     "terraform",
		IsActive: false,
	}

	data := flattenServiceAccountDataSource(account)

	if !data.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !data.CreatedAt.IsNull() {
		t.Fatal("expected created_at to be null")
	}
	if !data.CreatedBy.IsNull() {
		t.Fatal("expected created_by to be null")
	}
	if data.IsActive.ValueBool() {
		t.Fatal("expected is_active=false")
	}
}
