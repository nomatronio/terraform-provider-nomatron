package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewEnvironmentResource(t *testing.T) {
	t.Parallel()

	r := NewEnvironmentResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*EnvironmentResource); !ok {
		t.Fatalf("expected *EnvironmentResource, got %T", r)
	}
}

func TestEnvironmentResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &EnvironmentResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_environment" {
		t.Fatalf("expected type name %q, got %q", "nomatron_environment", resp.TypeName)
	}
}

func TestEnvironmentResource_Schema(t *testing.T) {
	t.Parallel()

	r := &EnvironmentResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "app_slug", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "job_slug", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "slug", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "cluster_id", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "namespace", true, false, false, false)
	assertResourceInt64Attribute(t, attrs, "priority", true, false, false)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "updated_at", false, false, true, false)
}

func TestEnvironmentResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &EnvironmentResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestEnvironmentResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &EnvironmentResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestStateFromEnvironment(t *testing.T) {
	t.Parallel()

	envID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 26, 16, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 26, 17, 0, 0, 0, time.UTC)

	state := stateFromEnvironment(EnvironmentResourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
		JobSlug: types.StringValue("web"),
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

	if state.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", state.OrgName.ValueString())
	}
	if state.AppSlug.ValueString() != "payments" {
		t.Fatalf("unexpected app_slug: %q", state.AppSlug.ValueString())
	}
	if state.JobSlug.ValueString() != "web" {
		t.Fatalf("unexpected job_slug: %q", state.JobSlug.ValueString())
	}
	if state.ID.ValueString() != envID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Slug.ValueString() != "prod" {
		t.Fatalf("unexpected slug: %q", state.Slug.ValueString())
	}
	if state.Name.ValueString() != "Production" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.ClusterID.ValueString() != clusterID.String() {
		t.Fatalf("unexpected cluster_id: %q", state.ClusterID.ValueString())
	}
	if state.Namespace.ValueString() != "payments-prod" {
		t.Fatalf("unexpected namespace: %q", state.Namespace.ValueString())
	}
	if state.Priority.ValueInt64() != 100 {
		t.Fatalf("unexpected priority: %d", state.Priority.ValueInt64())
	}
	if state.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", state.CreatedAt.ValueString())
	}
	if state.UpdatedAt.ValueString() != updatedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", state.UpdatedAt.ValueString())
	}
}

func TestParseEnvironmentImportID(t *testing.T) {
	t.Parallel()

	orgName, appSlug, jobSlug, environmentSlug, err := parseEnvironmentImportID("platform/payments/web/prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgName != "platform" || appSlug != "payments" || jobSlug != "web" || environmentSlug != "prod" {
		t.Fatalf("unexpected import id parts: %q %q %q %q", orgName, appSlug, jobSlug, environmentSlug)
	}
}

func TestParseEnvironmentImportID_Invalid(t *testing.T) {
	t.Parallel()

	if _, _, _, _, err := parseEnvironmentImportID("platform/payments/web"); err == nil {
		t.Fatal("expected invalid import id to fail")
	}
}

func TestEnvironmentNotFoundError(t *testing.T) {
	t.Parallel()

	err := &environmentNotFoundError{orgName: "platform", appSlug: "payments", jobSlug: "web", environmentSlug: "prod"}
	if !isEnvironmentNotFound(err) {
		t.Fatal("expected environmentNotFoundError to be recognized")
	}
	if isEnvironmentNotFound(errors.New("other")) {
		t.Fatal("expected non-environment error to be ignored")
	}
}

func TestBuildCreateEnvironmentBody(t *testing.T) {
	t.Parallel()

	body, diags := buildCreateEnvironmentBody(EnvironmentResourceModel{
		Name:      types.StringValue("Production"),
		Slug:      types.StringValue("prod"),
		ClusterID: types.StringValue("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Namespace: types.StringValue("payments-prod"),
		Priority:  types.Int64Value(100),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if body.Name != "Production" {
		t.Fatalf("unexpected name: %q", body.Name)
	}
	if body.Slug != "prod" {
		t.Fatalf("unexpected slug: %q", body.Slug)
	}
	if body.ClusterId.String() != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected cluster_id: %q", body.ClusterId.String())
	}
	if body.Namespace != "payments-prod" {
		t.Fatalf("unexpected namespace: %q", body.Namespace)
	}
	if body.Priority != 100 {
		t.Fatalf("unexpected priority: %d", body.Priority)
	}
}

func TestBuildUpdateEnvironmentBody(t *testing.T) {
	t.Parallel()

	body, diags := buildUpdateEnvironmentBody(
		EnvironmentResourceModel{
			Name:      types.StringValue("Production"),
			ClusterID: types.StringValue("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			Namespace: types.StringValue("payments-prod"),
			Priority:  types.Int64Value(100),
		},
		EnvironmentResourceModel{
			Name:      types.StringValue("Staging"),
			ClusterID: types.StringValue("cccccccc-cccc-cccc-cccc-cccccccccccc"),
			Namespace: types.StringValue("payments-staging"),
			Priority:  types.Int64Value(50),
		},
	)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if body.Name == nil || *body.Name != "Production" {
		t.Fatal("expected changed name to be set")
	}
	if body.ClusterId == nil || body.ClusterId.String() != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatal("expected changed cluster_id to be set")
	}
	if body.Namespace == nil || *body.Namespace != "payments-prod" {
		t.Fatal("expected changed namespace to be set")
	}
	if body.Priority == nil || *body.Priority != 100 {
		t.Fatal("expected changed priority to be set")
	}
}

func TestPriorityUpdateRequired(t *testing.T) {
	t.Parallel()

	if !priorityUpdateRequired(
		EnvironmentResourceModel{Priority: types.Int64Value(10)},
		sdk.Environment{Priority: 1},
	) {
		t.Fatal("expected differing priority to require update")
	}

	if priorityUpdateRequired(
		EnvironmentResourceModel{Priority: types.Int64Value(10)},
		sdk.Environment{Priority: 10},
	) {
		t.Fatal("expected matching priority to skip update")
	}

	if priorityUpdateRequired(
		EnvironmentResourceModel{Priority: types.Int64Unknown()},
		sdk.Environment{Priority: 1},
	) {
		t.Fatal("expected unknown priority to skip update")
	}
}

func TestInt64ValueChanged(t *testing.T) {
	t.Parallel()

	if !int64ValueChanged(types.Int64Value(1), types.Int64Value(2)) {
		t.Fatal("expected differing values to be treated as changed")
	}
	if int64ValueChanged(types.Int64Value(1), types.Int64Value(1)) {
		t.Fatal("expected identical values to be treated as unchanged")
	}
	if !int64ValueChanged(types.Int64Null(), types.Int64Value(1)) {
		t.Fatal("expected null/non-null difference to be treated as changed")
	}
}

func assertResourceInt64Attribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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
