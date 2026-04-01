package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewAgentDataSource(t *testing.T) {
	t.Parallel()

	ds := NewAgentDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*AgentDataSource); !ok {
		t.Fatalf("expected *AgentDataSource, got %T", ds)
	}
}

func TestAgentDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &AgentDataSource{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_network_agent" {
		t.Fatalf("expected type name %q, got %q", "nomatron_network_agent", resp.TypeName)
	}
}

func TestAgentDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &AgentDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertDataSourceStringAttribute(t, attrs, "id", false, true, true)
	assertDataSourceStringAttribute(t, attrs, "name", false, true, true)
	assertDataSourceStringAttribute(t, attrs, "description", false, false, true)
	assertDataSourceBoolAttribute(t, attrs, "is_active", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "created_at", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "created_by_type", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "created_by_id", false, false, true)
}

func TestAgentDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &AgentDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: nil,
	}, &resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestAgentDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &AgentDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestAgentDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &AgentDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: client,
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}

func TestFlattenAgentDataSource(t *testing.T) {
	t.Parallel()

	agentID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	userID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)
	description := "Private runner for edge workloads"

	agent := sdk.NetworkAgent{
		Id:              agentID,
		Name:            "edge-agent-1",
		Description:     &description,
		IsActive:        true,
		CreatedAt:       createdAt,
		CreatedByType:   sdk.NetworkAgentCreatedByTypeUser,
		CreatedByUserId: &userID,
	}

	data := flattenAgentDataSource(agent)

	if data.ID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "edge-agent-1" {
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
	if data.CreatedByType.ValueString() != "user" {
		t.Fatalf("unexpected created_by_type: %q", data.CreatedByType.ValueString())
	}
	if data.CreatedByID.ValueString() != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected created_by_id: %q", data.CreatedByID.ValueString())
	}
}

func TestFlattenAgentDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	agentID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))

	agent := sdk.NetworkAgent{
		Id:        agentID,
		Name:      "edge-agent-2",
		IsActive:  false,
		CreatedAt: time.Time{},
	}

	data := flattenAgentDataSource(agent)

	if !data.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !data.CreatedAt.IsNull() {
		t.Fatal("expected created_at to be null")
	}
	if !data.CreatedByType.IsNull() {
		t.Fatal("expected created_by_type to be null")
	}
	if !data.CreatedByID.IsNull() {
		t.Fatal("expected created_by_id to be null")
	}
}

func assertDataSourceStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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

func assertDataSourceBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	boolAttr, ok := attr.(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.BoolAttribute, got %T", name, attr)
	}

	if boolAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, boolAttr.Required)
	}
	if boolAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, boolAttr.Optional)
	}
	if boolAttr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, boolAttr.Computed)
	}
}

func assertDataSourceMapAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	mapAttr, ok := attr.(schema.MapAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.MapAttribute, got %T", name, attr)
	}

	if mapAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, mapAttr.Required)
	}
	if mapAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, mapAttr.Optional)
	}
	if mapAttr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, mapAttr.Computed)
	}
}
