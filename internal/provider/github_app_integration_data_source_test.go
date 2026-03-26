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

func TestNewGitHubAppIntegrationDataSource(t *testing.T) {
	t.Parallel()

	ds := NewGitHubAppIntegrationDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*GitHubAppIntegrationDataSource); !ok {
		t.Fatalf("expected *GitHubAppIntegrationDataSource, got %T", ds)
	}
}

func TestGitHubAppIntegrationDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &GitHubAppIntegrationDataSource{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_github_app_integration" {
		t.Fatalf("expected type name %q, got %q", "nomatron_github_app_integration", resp.TypeName)
	}
}

func TestGitHubAppIntegrationDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &GitHubAppIntegrationDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertGitHubIntegrationDataSourceStringAttribute(t, attrs, "name", true, false, false)
	assertGitHubIntegrationDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertGitHubIntegrationDataSourceStringAttribute(t, attrs, "app_id", false, false, true)
	assertGitHubIntegrationDataSourceStringAttribute(t, attrs, "app_slug", false, false, true)
	assertGitHubIntegrationDataSourceStringAttribute(t, attrs, "client_id", false, false, true)
	assertGitHubIntegrationDataSourceStringAttribute(t, attrs, "scope", false, false, true)
}

func TestGitHubAppIntegrationDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &GitHubAppIntegrationDataSource{}
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

func TestGitHubAppIntegrationDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &GitHubAppIntegrationDataSource{}
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

func TestGitHubAppIntegrationDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &GitHubAppIntegrationDataSource{}
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

func TestFlattenGitHubAppIntegrationDataSource(t *testing.T) {
	t.Parallel()

	connectionID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	appID := "12345"
	appSlug := "nomatron-app"
	clientID := "Iv1.1234567890abcdef"

	data := flattenGitHubAppIntegrationDataSource(GitHubAppIntegrationDataSourceModel{
		Name: types.StringValue("primary"),
	}, sdk.GitHubConnection{
		Id:       connectionID,
		Name:     "primary",
		AppId:    &appID,
		AppSlug:  &appSlug,
		ClientId: &clientID,
		Scope:    sdk.GitHubConnectionScopeGlobal,
	})

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
	if data.Scope.ValueString() != "global" {
		t.Fatalf("unexpected scope: %q", data.Scope.ValueString())
	}
}

func TestFlattenGitHubAppIntegrationDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	connectionID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))

	data := flattenGitHubAppIntegrationDataSource(GitHubAppIntegrationDataSourceModel{
		Name: types.StringValue("primary"),
	}, sdk.GitHubConnection{
		Id:    connectionID,
		Name:  "primary",
		Scope: sdk.GitHubConnectionScopeGlobal,
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
}

func assertGitHubIntegrationDataSourceStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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
