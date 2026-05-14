package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewOrganizationGitHubAppIntegrationResource(t *testing.T) {
	t.Parallel()

	r := NewOrganizationGitHubAppIntegrationResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*OrganizationGitHubAppIntegrationResource); !ok {
		t.Fatalf("expected *OrganizationGitHubAppIntegrationResource, got %T", r)
	}
}

func TestOrganizationGitHubAppIntegrationResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &OrganizationGitHubAppIntegrationResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_organization_github_app_integration" {
		t.Fatalf("expected type name %q, got %q", "nomatron_organization_github_app_integration", resp.TypeName)
	}
}

func TestOrganizationGitHubAppIntegrationResource_Schema(t *testing.T) {
	t.Parallel()

	r := &OrganizationGitHubAppIntegrationResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "provider_kind", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "web_base_url", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "api_base_url", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "upload_base_url", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "app_id", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "app_slug", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "client_id", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "private_key_pem_wo", true, false, false, true)
	assertResourceStringAttribute(t, attrs, "webhook_secret_wo", true, false, false, true)
	assertResourceStringAttribute(t, attrs, "tls_ca_bundle_pem", false, true, false, true)
	assertResourceStringAttribute(t, attrs, "scope", false, false, true, false)
}

func TestOrganizationGitHubAppIntegrationResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &OrganizationGitHubAppIntegrationResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestOrganizationGitHubAppIntegrationResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &OrganizationGitHubAppIntegrationResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestStateFromOrganizationGitHubConnection(t *testing.T) {
	t.Parallel()

	connectionID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	appID := "12345"
	appSlug := "nomatron-app"
	clientID := "Iv1.1234567890abcdef"
	webBaseURL := "https://github.enterprise.example"
	apiBaseURL := "https://github.enterprise.example/api/v3"
	uploadBaseURL := "https://github.enterprise.example/api/uploads"
	providerKind := sdk.EnterpriseServer

	state := stateFromOrganizationGitHubConnection(OrganizationGitHubAppIntegrationResourceModel{
		OrgName: types.StringValue("platform"),
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

	if state.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", state.OrgName.ValueString())
	}
	if state.ID.ValueString() != connectionID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "primary" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.AppID.ValueString() != appID {
		t.Fatalf("unexpected app_id: %q", state.AppID.ValueString())
	}
	if state.AppSlug.ValueString() != appSlug {
		t.Fatalf("unexpected app_slug: %q", state.AppSlug.ValueString())
	}
	if state.ClientID.ValueString() != clientID {
		t.Fatalf("unexpected client_id: %q", state.ClientID.ValueString())
	}
	if state.ProviderKind.ValueString() != string(sdk.EnterpriseServer) {
		t.Fatalf("unexpected provider_kind: %q", state.ProviderKind.ValueString())
	}
	if state.WebBaseURL.ValueString() != webBaseURL {
		t.Fatalf("unexpected web_base_url: %q", state.WebBaseURL.ValueString())
	}
	if state.APIBaseURL.ValueString() != apiBaseURL {
		t.Fatalf("unexpected api_base_url: %q", state.APIBaseURL.ValueString())
	}
	if state.UploadBaseURL.ValueString() != uploadBaseURL {
		t.Fatalf("unexpected upload_base_url: %q", state.UploadBaseURL.ValueString())
	}
	if state.Scope.ValueString() != "org" {
		t.Fatalf("unexpected scope: %q", state.Scope.ValueString())
	}
}

func TestOrganizationGitHubIntegrationNotFoundError(t *testing.T) {
	t.Parallel()

	err := &organizationGitHubIntegrationNotFoundError{orgName: "platform", name: "primary"}
	if !isOrganizationGitHubIntegrationNotFound(err) {
		t.Fatal("expected organizationGitHubIntegrationNotFoundError to be recognized")
	}
	if isOrganizationGitHubIntegrationNotFound(errors.New("other")) {
		t.Fatal("expected non-integration error to be ignored")
	}
}
