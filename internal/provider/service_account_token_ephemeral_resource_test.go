package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	helperresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

func TestNewServiceAccountTokenEphemeralResource(t *testing.T) {
	t.Parallel()

	r := NewServiceAccountTokenEphemeralResource()
	if r == nil {
		t.Fatal("expected ephemeral resource to be non-nil")
	}

	if _, ok := r.(*ServiceAccountTokenEphemeralResource); !ok {
		t.Fatalf("expected *ServiceAccountTokenEphemeralResource, got %T", r)
	}
}

func TestServiceAccountTokenEphemeralResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &ServiceAccountTokenEphemeralResource{}
	req := ephemeral.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	resp := &ephemeral.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_service_account_token" {
		t.Fatalf("expected type name %q, got %q", "nomatron_service_account_token", resp.TypeName)
	}
}

func TestServiceAccountTokenEphemeralResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &ServiceAccountTokenEphemeralResource{}
	resp := &ephemeral.ConfigureResponse{}

	r.Configure(context.Background(), ephemeral.ConfigureRequest{
		ProviderData: nil,
	}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestServiceAccountTokenEphemeralResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &ServiceAccountTokenEphemeralResource{}
	resp := &ephemeral.ConfigureResponse{}

	r.Configure(context.Background(), ephemeral.ConfigureRequest{
		ProviderData: "not-a-client",
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestServiceAccountTokenEphemeralResource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	r := &ServiceAccountTokenEphemeralResource{}
	resp := &ephemeral.ConfigureResponse{}

	r.Configure(context.Background(), ephemeral.ConfigureRequest{
		ProviderData: client,
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if r.client != client {
		t.Fatal("expected configured client to be stored on ephemeral resource")
	}
}

func TestServiceAccountTokenEphemeralResource_EchoProvider(t *testing.T) {
	t.Parallel()

	serviceAccountID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	tokenID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	requestID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	expiresAt := time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC)
	requestTime := time.Date(2026, 3, 26, 18, 0, 0, 0, time.UTC)

	server := newTestHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/v1/service-accounts/"+serviceAccountID.String()+"/tokens" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer service-account-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"service_account_id": serviceAccountID.String(),
				"token":              "nomatron_service_account_secret",
				"token_id":           tokenID.String(),
				"token_prefix":       "nomatron_sat",
				"expires_at":         expiresAt.Format(time.RFC3339),
			},
			"errors": []interface{}{},
			"meta": map[string]interface{}{
				"code":         "CREATED",
				"request_id":   requestID.String(),
				"request_time": requestTime.Format(time.RFC3339),
				"status":       "success",
			},
			"warnings": []interface{}{},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	helperresource.UnitTest(t, helperresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"nomatron": providerserver.NewProtocol6WithError(New("test")()),
			"echo":     echoprovider.NewProviderServer(),
		},
		Steps: []helperresource.TestStep{
			{
				Config: `
provider "nomatron" {
  address = "` + server.URL + `"
  token   = "service-account-token"
}

ephemeral "nomatron_service_account_token" "test" {
  service_account_id = "` + serviceAccountID.String() + `"
  name               = "terraform"
  expires_at         = "` + expiresAt.Format(time.RFC3339) + `"
}

provider "echo" {
  data = ephemeral.nomatron_service_account_token.test
}

resource "echo" "test" {}
`,
				ConfigPlanChecks: helperresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue("echo.test", tfjsonpath.New("data")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("service_account_id"), knownvalue.StringExact(serviceAccountID.String())),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("token_id"), knownvalue.StringExact(tokenID.String())),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("token_prefix"), knownvalue.StringExact("nomatron_sat")),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("expires_at"), knownvalue.StringExact(expiresAt.Format(time.RFC3339))),
				},
			},
		},
	})
}
