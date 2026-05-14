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

func TestNewOrganizationGitHubAppIntegrationDataSource(t *testing.T) {
	t.Parallel()

	ds := NewOrganizationGitHubAppIntegrationDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*OrganizationGitHubAppIntegrationDataSource); !ok {
		t.Fatalf("expected *OrganizationGitHubAppIntegrationDataSource, got %T", ds)
	}
}

func TestOrganizationGitHubAppIntegrationDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &OrganizationGitHubAppIntegrationDataSource{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_organization_github_app_integration" {
		t.Fatalf("expected type name %q, got %q", "nomatron_organization_github_app_integration", resp.TypeName)
	}
}

func TestOrganizationGitHubAppIntegrationDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &OrganizationGitHubAppIntegrationDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "name", true, false, false)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "provider_kind", false, false, true)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "web_base_url", false, false, true)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "api_base_url", false, false, true)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "upload_base_url", false, false, true)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "app_id", false, false, true)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "app_slug", false, false, true)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "client_id", false, false, true)
	assertOrganizationGitHubIntegrationDataSourceStringAttribute(t, attrs, "scope", false, false, true)
}

func TestOrganizationGitHubAppIntegrationDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &OrganizationGitHubAppIntegrationDataSource{}
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

func TestOrganizationGitHubAppIntegrationDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &OrganizationGitHubAppIntegrationDataSource{}
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

func TestOrganizationGitHubAppIntegrationDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &OrganizationGitHubAppIntegrationDataSource{}
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

func TestFlattenOrganizationGitHubAppIntegrationDataSource(t *testing.T) {
	t.Parallel()

	connectionID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	appID := "12345"
	appSlug := "nomatron-app"
	clientID := "Iv1.1234567890abcdef"
	webBaseURL := "https://github.enterprise.example"
	apiBaseURL := "https://github.enterprise.example/api/v3"
	uploadBaseURL := "https://github.enterprise.example/api/uploads"
	providerKind := sdk.EnterpriseServer

	data := flattenOrganizationGitHubAppIntegrationDataSource(OrganizationGitHubAppIntegrationDataSourceModel{
		OrgName: types.StringValue("platform"),
		Name:    types.StringValue("primary"),
	}, sdk.GitHubConnection{
		Id:            connectionID,
		Name:          "primary",
		AppId:         &appID,
		AppSlug:       &appSlug,
		ClientId:      &clientID,
		ProviderKind:  &providerKind,
		WebBaseUrl:    &webBaseURL,
		ApiBaseUrl:    &apiBaseURL,
		UploadBaseUrl: &uploadBaseURL,
		Scope:         sdk.GitHubConnectionScopeOrg,
	})

	if data.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", data.OrgName.ValueString())
	}
	if data.Name.ValueString() != "primary" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.ID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.AppID.ValueString() != appID {
		t.Fatalf("unexpected app_id: %q", data.AppID.ValueString())
	}
	if data.AppSlug.ValueString() != appSlug {
		t.Fatalf("unexpected app_slug: %q", data.AppSlug.ValueString())
	}
	if data.ClientID.ValueString() != clientID {
		t.Fatalf("unexpected client_id: %q", data.ClientID.ValueString())
	}
	if data.ProviderKind.ValueString() != string(sdk.EnterpriseServer) {
		t.Fatalf("unexpected provider_kind: %q", data.ProviderKind.ValueString())
	}
	if data.WebBaseURL.ValueString() != webBaseURL {
		t.Fatalf("unexpected web_base_url: %q", data.WebBaseURL.ValueString())
	}
	if data.APIBaseURL.ValueString() != apiBaseURL {
		t.Fatalf("unexpected api_base_url: %q", data.APIBaseURL.ValueString())
	}
	if data.UploadBaseURL.ValueString() != uploadBaseURL {
		t.Fatalf("unexpected upload_base_url: %q", data.UploadBaseURL.ValueString())
	}
	if data.Scope.ValueString() != "org" {
		t.Fatalf("unexpected scope: %q", data.Scope.ValueString())
	}
}

func TestFlattenOrganizationGitHubAppIntegrationDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	connectionID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))

	data := flattenOrganizationGitHubAppIntegrationDataSource(OrganizationGitHubAppIntegrationDataSourceModel{
		OrgName: types.StringValue("platform"),
		Name:    types.StringValue("primary"),
	}, sdk.GitHubConnection{
		Id:    connectionID,
		Name:  "primary",
		Scope: sdk.GitHubConnectionScopeOrg,
	})

	if !data.AppID.IsNull() {
		t.Fatal("expected app_id to be null")
	}
	if !data.AppSlug.IsNull() {
		t.Fatal("expected app_slug to be null")
	}
	if !data.ClientID.IsNull() {
		t.Fatal("expected client_id to be null")
	}
	if !data.ProviderKind.IsNull() {
		t.Fatal("expected provider_kind to be null")
	}
	if !data.WebBaseURL.IsNull() {
		t.Fatal("expected web_base_url to be null")
	}
	if !data.APIBaseURL.IsNull() {
		t.Fatal("expected api_base_url to be null")
	}
	if !data.UploadBaseURL.IsNull() {
		t.Fatal("expected upload_base_url to be null")
	}
}

func assertOrganizationGitHubIntegrationDataSourceStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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
