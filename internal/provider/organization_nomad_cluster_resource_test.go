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

func TestNewOrganizationNomadClusterResource(t *testing.T) {
	t.Parallel()

	r := NewOrganizationNomadClusterResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*OrganizationNomadClusterResource); !ok {
		t.Fatalf("expected *OrganizationNomadClusterResource, got %T", r)
	}
}

func TestOrganizationNomadClusterResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &OrganizationNomadClusterResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_organization_nomad_cluster" {
		t.Fatalf("expected type name %q, got %q", "nomatron_organization_nomad_cluster", resp.TypeName)
	}
}

func TestOrganizationNomadClusterResource_Schema(t *testing.T) {
	t.Parallel()

	r := &OrganizationNomadClusterResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "connectivity_mode", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "address", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "agent_id", false, true, false, false)
	assertResourceBoolAttribute(t, attrs, "skip_verify", false, true, true)
	assertResourceStringAttribute(t, attrs, "acl_token_wo", false, true, false, true)
	assertResourceStringAttribute(t, attrs, "ca_cert_wo", false, true, false, true)
	assertResourceStringAttribute(t, attrs, "tls_cert_wo", false, true, false, true)
	assertResourceStringAttribute(t, attrs, "tls_key_wo", false, true, false, true)
	assertResourceStringAttribute(t, attrs, "scope", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "updated_at", false, false, true, false)
}

func TestOrganizationNomadClusterResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &OrganizationNomadClusterResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestOrganizationNomadClusterResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &OrganizationNomadClusterResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestStateFromOrganizationNomadCluster(t *testing.T) {
	t.Parallel()

	clusterID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	agentID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	description := "Primary org nomad cluster"
	address := "https://nomad.example.com"

	base := OrganizationNomadClusterResourceModel{
		OrgName: types.StringValue("platform"),
	}
	cluster := sdk.Cluster{
		Id:               clusterID,
		Name:             "primary",
		Description:      &description,
		Address:          &address,
		NetworkAgentId:   &agentID,
		ConnectivityMode: sdk.ClusterConnectivityModeDirect,
		Scope:            "cccccccc-cccc-cccc-cccc-cccccccccccc",
		SkipVerify:       true,
	}

	state := stateFromOrganizationNomadCluster(base, cluster)

	if state.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", state.OrgName.ValueString())
	}
	if state.ID.ValueString() != clusterID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "primary" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if state.Address.ValueString() != address {
		t.Fatalf("unexpected address: %q", state.Address.ValueString())
	}
	if state.AgentID.ValueString() != agentID.String() {
		t.Fatalf("unexpected agent_id: %q", state.AgentID.ValueString())
	}
	if state.Scope.ValueString() != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Fatalf("unexpected scope: %q", state.Scope.ValueString())
	}
	if !state.SkipVerify.ValueBool() {
		t.Fatal("expected skip_verify=true")
	}
}

func TestOrganizationNomadClusterNotFoundError(t *testing.T) {
	t.Parallel()

	err := &organizationNomadClusterNotFoundError{orgName: "platform", name: "primary"}
	if !isOrganizationNomadClusterNotFound(err) {
		t.Fatal("expected organizationNomadClusterNotFoundError to be recognized")
	}
	if isOrganizationNomadClusterNotFound(errors.New("other")) {
		t.Fatal("expected non-cluster error to be ignored")
	}
}
