package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewApplicationDataSource(t *testing.T) {
	t.Parallel()

	ds := NewApplicationDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*ApplicationDataSource); !ok {
		t.Fatalf("expected *ApplicationDataSource, got %T", ds)
	}
}

func TestApplicationDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &ApplicationDataSource{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_application" {
		t.Fatalf("expected type name %q, got %q", "nomatron_application", resp.TypeName)
	}
}

func TestApplicationDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &ApplicationDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertApplicationDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertApplicationDataSourceStringAttribute(t, attrs, "slug", true, false, false)
	assertApplicationDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "name", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "description", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "cluster_id", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "repo_url", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "git_provider", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "ref", false, false, true)
	assertApplicationDataSourceBoolAttribute(t, attrs, "auto_plan", false, false, true)
	assertApplicationDataSourceBoolAttribute(t, attrs, "auto_apply", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "vcs_github_id", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "created_at", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "updated_at", false, false, true)
	assertApplicationDataSourceStringAttribute(t, attrs, "updated_by", false, false, true)
}

func TestApplicationDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &ApplicationDataSource{}
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

func TestApplicationDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &ApplicationDataSource{}
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

func TestApplicationDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &ApplicationDataSource{}
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

func TestFlattenApplicationDataSource(t *testing.T) {
	t.Parallel()

	appID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	vcsGitHubID := openapi_types.UUID(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"))
	updatedBy := openapi_types.UUID(uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"))
	description := "primary app"
	ref := "main"
	createdAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 26, 13, 0, 0, 0, time.UTC)

	data := flattenApplicationDataSource(ApplicationDataSourceModel{
		OrgName: types.StringValue("platform"),
		Slug:    types.StringValue("payments"),
	}, sdk.App{
		Id:          appID,
		Slug:        "payments",
		Name:        "Payments",
		Description: &description,
		ClusterId:   clusterID,
		RepoUrl:     "https://github.com/nomatron/payments",
		GitProvider: sdk.Github,
		Ref:         &ref,
		AutoPlan:    true,
		AutoApply:   false,
		VcsGithubId: &vcsGitHubID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		UpdatedBy:   &updatedBy,
	})

	if data.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", data.OrgName.ValueString())
	}
	if data.Slug.ValueString() != "payments" {
		t.Fatalf("unexpected slug: %q", data.Slug.ValueString())
	}
	if data.ID.ValueString() != appID.String() {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "Payments" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", data.Description.ValueString())
	}
	if data.ClusterID.ValueString() != clusterID.String() {
		t.Fatalf("unexpected cluster_id: %q", data.ClusterID.ValueString())
	}
	if data.RepoURL.ValueString() != "https://github.com/nomatron/payments" {
		t.Fatalf("unexpected repo_url: %q", data.RepoURL.ValueString())
	}
	if data.GitProvider.ValueString() != "github" {
		t.Fatalf("unexpected git_provider: %q", data.GitProvider.ValueString())
	}
	if data.Ref.ValueString() != ref {
		t.Fatalf("unexpected ref: %q", data.Ref.ValueString())
	}
	if !data.AutoPlan.ValueBool() {
		t.Fatal("expected auto_plan=true")
	}
	if data.AutoApply.ValueBool() {
		t.Fatal("expected auto_apply=false")
	}
	if data.VcsGitHubID.ValueString() != vcsGitHubID.String() {
		t.Fatalf("unexpected vcs_github_id: %q", data.VcsGitHubID.ValueString())
	}
	if data.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", data.CreatedAt.ValueString())
	}
	if data.UpdatedAt.ValueString() != updatedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", data.UpdatedAt.ValueString())
	}
	if data.UpdatedBy.ValueString() != updatedBy.String() {
		t.Fatalf("unexpected updated_by: %q", data.UpdatedBy.ValueString())
	}
}

func TestFlattenApplicationDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	appID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))

	data := flattenApplicationDataSource(ApplicationDataSourceModel{
		OrgName: types.StringValue("platform"),
		Slug:    types.StringValue("payments"),
	}, sdk.App{
		Id:          appID,
		Slug:        "payments",
		Name:        "Payments",
		ClusterId:   clusterID,
		RepoUrl:     "https://github.com/nomatron/payments",
		GitProvider: sdk.Github,
		AutoPlan:    false,
		AutoApply:   false,
	})

	if !data.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !data.Ref.IsNull() {
		t.Fatal("expected ref to be null")
	}
	if !data.VcsGitHubID.IsNull() {
		t.Fatal("expected vcs_github_id to be null")
	}
	if !data.CreatedAt.IsNull() {
		t.Fatal("expected created_at to be null")
	}
	if !data.UpdatedAt.IsNull() {
		t.Fatal("expected updated_at to be null")
	}
	if !data.UpdatedBy.IsNull() {
		t.Fatal("expected updated_by to be null")
	}
}

func assertApplicationDataSourceStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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

func assertApplicationDataSourceBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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
