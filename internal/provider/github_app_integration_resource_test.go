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

func TestNewGitHubAppIntegrationResource(t *testing.T) {
	t.Parallel()

	r := NewGitHubAppIntegrationResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*GitHubAppIntegrationResource); !ok {
		t.Fatalf("expected *GitHubAppIntegrationResource, got %T", r)
	}
}

func TestGitHubAppIntegrationResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &GitHubAppIntegrationResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_github_app_integration" {
		t.Fatalf("expected type name %q, got %q", "nomatron_github_app_integration", resp.TypeName)
	}
}

func TestGitHubAppIntegrationResource_Schema(t *testing.T) {
	t.Parallel()

	r := &GitHubAppIntegrationResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "app_id", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "app_slug", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "client_id", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "private_key_pem", true, false, false, true)
	assertResourceStringAttribute(t, attrs, "webhook_secret", true, false, false, true)
	assertResourceStringAttribute(t, attrs, "scope", false, false, true, false)
}

func TestGitHubAppIntegrationResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &GitHubAppIntegrationResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestGitHubAppIntegrationResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &GitHubAppIntegrationResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestStateFromGitHubConnection(t *testing.T) {
	t.Parallel()

	connectionID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	appID := "12345"
	appSlug := "nomatron-app"
	clientID := "Iv1.1234567890abcdef"

	state := stateFromGitHubConnection(GitHubAppIntegrationResourceModel{}, sdk.GitHubConnection{
		Id:       connectionID,
		Name:     "primary",
		AppId:    &appID,
		AppSlug:  &appSlug,
		ClientId: &clientID,
		Scope:    sdk.GitHubConnectionScopeGlobal,
	})

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
	if state.Scope.ValueString() != "global" {
		t.Fatalf("unexpected scope: %q", state.Scope.ValueString())
	}
}

func TestStateFromGitHubConnection_PreservesBaseFields(t *testing.T) {
	t.Parallel()

	connectionID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))

	state := stateFromGitHubConnection(GitHubAppIntegrationResourceModel{
		AppID:    types.StringValue("12345"),
		AppSlug:  types.StringValue("nomatron-app"),
		ClientID: types.StringValue("Iv1.1234567890abcdef"),
	}, sdk.GitHubConnection{
		Id:    connectionID,
		Name:  "primary",
		Scope: sdk.GitHubConnectionScopeGlobal,
	})

	if state.AppID.ValueString() != "12345" {
		t.Fatalf("unexpected app_id: %q", state.AppID.ValueString())
	}
	if state.AppSlug.ValueString() != "nomatron-app" {
		t.Fatalf("unexpected app_slug: %q", state.AppSlug.ValueString())
	}
	if state.ClientID.ValueString() != "Iv1.1234567890abcdef" {
		t.Fatalf("unexpected client_id: %q", state.ClientID.ValueString())
	}
}

func TestGitHubIntegrationNotFoundError(t *testing.T) {
	t.Parallel()

	err := &gitHubIntegrationNotFoundError{name: "primary"}
	if !isGitHubIntegrationNotFound(err) {
		t.Fatal("expected gitHubIntegrationNotFoundError to be recognized")
	}
	if isGitHubIntegrationNotFound(errors.New("other")) {
		t.Fatal("expected non-integration error to be ignored")
	}
}
