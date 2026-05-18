package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewEnvironmentDataSource(t *testing.T) {
	t.Parallel()

	ds := NewEnvironmentDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*EnvironmentDataSource); !ok {
		t.Fatalf("expected *EnvironmentDataSource, got %T", ds)
	}
}

func TestEnvironmentDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &EnvironmentDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "nomatron"}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_environment" {
		t.Fatalf("expected type name %q, got %q", "nomatron_environment", resp.TypeName)
	}
}

func TestEnvironmentDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &EnvironmentDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertEnvironmentDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertEnvironmentDataSourceStringAttribute(t, attrs, "app_slug", true, false, false)
	assertEnvironmentDataSourceStringAttribute(t, attrs, "slug", true, false, false)
	assertEnvironmentDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertEnvironmentDataSourceStringAttribute(t, attrs, "name", false, false, true)
	assertEnvironmentDataSourceStringAttribute(t, attrs, "cluster_id", false, false, true)
	assertEnvironmentDataSourceStringAttribute(t, attrs, "namespace", false, false, true)
	assertEnvironmentDataSourceInt64Attribute(t, attrs, "priority", false, false, true)
	assertEnvironmentDataSourceStringAttribute(t, attrs, "created_at", false, false, true)
	assertEnvironmentDataSourceStringAttribute(t, attrs, "updated_at", false, false, true)
}

func TestEnvironmentDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &EnvironmentDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, &resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestEnvironmentDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &EnvironmentDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "not-a-client"}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestEnvironmentDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &EnvironmentDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}

func TestFlattenEnvironmentDataSource(t *testing.T) {
	t.Parallel()

	envID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 26, 16, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 26, 17, 0, 0, 0, time.UTC)

	data := flattenEnvironmentDataSource(EnvironmentDataSourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
		Slug:    types.StringValue("prod"),
	}, sdk.Environment{
		Id:        envID,
		Slug:      "prod",
		Name:      "Production",
		ClusterId: clusterID,
		Namespace: "payments-prod",
		Priority:  100,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})

	if data.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", data.OrgName.ValueString())
	}
	if data.AppSlug.ValueString() != "payments" {
		t.Fatalf("unexpected app_slug: %q", data.AppSlug.ValueString())
	}
	if data.Slug.ValueString() != "prod" {
		t.Fatalf("unexpected slug: %q", data.Slug.ValueString())
	}
	if data.ID.ValueString() != envID.String() {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "Production" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.ClusterID.ValueString() != clusterID.String() {
		t.Fatalf("unexpected cluster_id: %q", data.ClusterID.ValueString())
	}
	if data.Namespace.ValueString() != "payments-prod" {
		t.Fatalf("unexpected namespace: %q", data.Namespace.ValueString())
	}
	if data.Priority.ValueInt64() != 100 {
		t.Fatalf("unexpected priority: %d", data.Priority.ValueInt64())
	}
	if data.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", data.CreatedAt.ValueString())
	}
	if data.UpdatedAt.ValueString() != updatedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", data.UpdatedAt.ValueString())
	}
}

func TestFlattenEnvironmentDataSource_WithZeroTimes(t *testing.T) {
	t.Parallel()

	envID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))

	data := flattenEnvironmentDataSource(EnvironmentDataSourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
		Slug:    types.StringValue("prod"),
	}, sdk.Environment{
		Id:        envID,
		Slug:      "prod",
		Name:      "Production",
		ClusterId: clusterID,
		Namespace: "payments-prod",
		Priority:  100,
	})

	if !data.CreatedAt.IsNull() {
		t.Fatal("expected created_at to be null")
	}
	if !data.UpdatedAt.IsNull() {
		t.Fatal("expected updated_at to be null")
	}
}

func assertEnvironmentDataSourceStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	stringAttr, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.StringAttribute, got %T", name, attr)
	}

	if stringAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, stringAttr.Required)
	}
	if stringAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, stringAttr.Optional)
	}
	if stringAttr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, stringAttr.Computed)
	}
}

func assertEnvironmentDataSourceInt64Attribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	int64Attr, ok := attr.(schema.Int64Attribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.Int64Attribute, got %T", name, attr)
	}

	if int64Attr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, int64Attr.Required)
	}
	if int64Attr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, int64Attr.Optional)
	}
	if int64Attr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, int64Attr.Computed)
	}
}
