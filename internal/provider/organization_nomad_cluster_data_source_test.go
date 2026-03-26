package provider

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewOrganizationNomadClusterDataSource(t *testing.T) {
	t.Parallel()

	ds := NewOrganizationNomadClusterDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*OrganizationNomadClusterDataSource); !ok {
		t.Fatalf("expected *OrganizationNomadClusterDataSource, got %T", ds)
	}
}

func TestOrganizationNomadClusterDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &OrganizationNomadClusterDataSource{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_organization_nomad_cluster" {
		t.Fatalf("expected type name %q, got %q", "nomatron_organization_nomad_cluster", resp.TypeName)
	}
}

func TestOrganizationNomadClusterDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &OrganizationNomadClusterDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertOrganizationNomadClusterDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertOrganizationNomadClusterDataSourceStringAttribute(t, attrs, "name", true, false, false)
	assertOrganizationNomadClusterDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertOrganizationNomadClusterDataSourceStringAttribute(t, attrs, "description", false, false, true)
	assertOrganizationNomadClusterDataSourceStringAttribute(t, attrs, "connectivity_mode", false, false, true)
	assertOrganizationNomadClusterDataSourceStringAttribute(t, attrs, "address", false, false, true)
	assertOrganizationNomadClusterDataSourceStringAttribute(t, attrs, "agent_id", false, false, true)
	assertOrganizationNomadClusterDataSourceBoolAttribute(t, attrs, "skip_verify", false, false, true)
	assertOrganizationNomadClusterDataSourceStringAttribute(t, attrs, "scope", false, false, true)
}

func TestOrganizationNomadClusterDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &OrganizationNomadClusterDataSource{}
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

func TestOrganizationNomadClusterDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &OrganizationNomadClusterDataSource{}
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

func TestOrganizationNomadClusterDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &OrganizationNomadClusterDataSource{}
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

func TestFlattenOrganizationNomadClusterDataSource(t *testing.T) {
	t.Parallel()

	clusterID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	agentID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	description := "primary org cluster"
	address := "https://nomad.example.com"

	cluster := sdk.Cluster{
		Id:               clusterID,
		Name:             "prod-primary",
		Description:      &description,
		ConnectivityMode: sdk.ClusterConnectivityModeDirect,
		Address:          &address,
		AgentId:          &agentID,
		SkipVerify:       true,
		Scope:            "cccccccc-cccc-cccc-cccc-cccccccccccc",
	}

	data := flattenOrganizationNomadClusterDataSource(OrganizationNomadClusterDataSourceModel{
		OrgName: types.StringValue("platform"),
		Name:    types.StringValue("prod-primary"),
	}, cluster)

	if data.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", data.OrgName.ValueString())
	}
	if data.Name.ValueString() != "prod-primary" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.ID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", data.Description.ValueString())
	}
	if data.ConnectivityMode.ValueString() != "direct" {
		t.Fatalf("unexpected connectivity_mode: %q", data.ConnectivityMode.ValueString())
	}
	if data.Address.ValueString() != address {
		t.Fatalf("unexpected address: %q", data.Address.ValueString())
	}
	if data.AgentID.ValueString() != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected agent_id: %q", data.AgentID.ValueString())
	}
	if !data.SkipVerify.ValueBool() {
		t.Fatal("expected skip_verify=true")
	}
	if data.Scope.ValueString() != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Fatalf("unexpected scope: %q", data.Scope.ValueString())
	}
}

func TestFlattenOrganizationNomadClusterDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	clusterID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))

	cluster := sdk.Cluster{
		Id:               clusterID,
		Name:             "prod-primary",
		ConnectivityMode: sdk.ClusterConnectivityModeAgent,
		SkipVerify:       false,
		Scope:            "cccccccc-cccc-cccc-cccc-cccccccccccc",
	}

	data := flattenOrganizationNomadClusterDataSource(OrganizationNomadClusterDataSourceModel{
		OrgName: types.StringValue("platform"),
		Name:    types.StringValue("prod-primary"),
	}, cluster)

	if !data.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !data.Address.IsNull() {
		t.Fatal("expected address to be null")
	}
	if !data.AgentID.IsNull() {
		t.Fatal("expected agent_id to be null")
	}
}

func assertOrganizationNomadClusterDataSourceStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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

func assertOrganizationNomadClusterDataSourceBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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
