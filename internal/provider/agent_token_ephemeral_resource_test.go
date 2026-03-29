package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func newTestHTTPServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	var server *httptest.Server

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("skipping http-backed ephemeral test because local listeners are unavailable in this environment: %v", r)
			}
		}()

		server = httptest.NewServer(handler)
	}()

	if server == nil {
		t.Skip("skipping http-backed ephemeral test because local listeners are unavailable in this environment")
	}

	return server
}

func TestNewAgentTokenEphemeralResource(t *testing.T) {
	t.Parallel()

	r := NewAgentTokenEphemeralResource()
	if r == nil {
		t.Fatal("expected ephemeral resource to be non-nil")
	}

	if _, ok := r.(*AgentTokenEphemeralResource); !ok {
		t.Fatalf("expected *AgentTokenEphemeralResource, got %T", r)
	}
}

func TestAgentTokenEphemeralResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &AgentTokenEphemeralResource{}
	req := ephemeral.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	resp := &ephemeral.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_agent_token" {
		t.Fatalf("expected type name %q, got %q", "nomatron_agent_token", resp.TypeName)
	}
}

func TestAgentTokenEphemeralResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &AgentTokenEphemeralResource{}
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

func TestAgentTokenEphemeralResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &AgentTokenEphemeralResource{}
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

func TestAgentTokenEphemeralResource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	r := &AgentTokenEphemeralResource{}
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

func TestAgentTokenEphemeralResource_EchoProvider(t *testing.T) {
	t.Parallel()

	agentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	tokenID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	requestID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	rotatedAt := time.Date(2026, 3, 26, 18, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 3, 26, 17, 0, 0, 0, time.UTC)

	server := newTestHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		if req.URL.Path != "/api/v1/network-agents/"+agentID.String()+"/tokens/rotate" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer service-account-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"agent": map[string]interface{}{
					"id":              agentID.String(),
					"name":            "test-agent",
					"is_active":       true,
					"created_at":      createdAt.Format(time.RFC3339),
					"updated_at":      createdAt.Format(time.RFC3339),
					"created_by_type": "user",
				},
				"minted": map[string]interface{}{
					"token":        "nomatron_agent_secret",
					"token_id":     tokenID.String(),
					"token_prefix": "nomatron_agent",
				},
				"revoked_tokens": 1,
				"rotated_at":     rotatedAt.Format(time.RFC3339),
			},
			"errors": []interface{}{},
			"meta": map[string]interface{}{
				"code":         "OK",
				"request_id":   requestID.String(),
				"request_time": rotatedAt.Format(time.RFC3339),
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

ephemeral "nomatron_agent_token" "test" {
  agent_id        = "` + agentID.String() + `"
  name            = "terraform"
  revoke_existing = true
}

provider "echo" {
  data = ephemeral.nomatron_agent_token.test
}

resource "echo" "test" {}
`,
				ConfigPlanChecks: helperresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue("echo.test", tfjsonpath.New("data")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("agent_id"), knownvalue.StringExact(agentID.String())),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("token_id"), knownvalue.StringExact(tokenID.String())),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("token_prefix"), knownvalue.StringExact("nomatron_agent")),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("revoked_tokens"), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue("echo.test", tfjsonpath.New("data").AtMapKey("rotated_at"), knownvalue.StringExact(rotatedAt.Format(time.RFC3339))),
				},
			},
		},
	})
}
