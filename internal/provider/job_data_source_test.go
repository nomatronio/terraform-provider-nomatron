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

func TestNewJobDataSource(t *testing.T) {
	t.Parallel()

	ds := NewJobDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*JobDataSource); !ok {
		t.Fatalf("expected *JobDataSource, got %T", ds)
	}
}

func TestJobDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &JobDataSource{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_job" {
		t.Fatalf("expected type name %q, got %q", "nomatron_job", resp.TypeName)
	}
}

func TestJobDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &JobDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertJobDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertJobDataSourceStringAttribute(t, attrs, "app_slug", true, false, false)
	assertJobDataSourceStringAttribute(t, attrs, "slug", true, false, false)
	assertJobDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "name", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "description", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "cluster_id", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "default_namespace", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "repo_url", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "effective_repo_url", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "jobspec_path", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "jobspec_type", false, false, true)
	assertJobDataSourceBoolAttribute(t, attrs, "is_primary", false, false, true)
	assertJobDataSourceInt64Attribute(t, attrs, "priority", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "created_at", false, false, true)
	assertJobDataSourceStringAttribute(t, attrs, "updated_at", false, false, true)
}

func TestJobDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &JobDataSource{}
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

func TestJobDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &JobDataSource{}
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

func TestJobDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &JobDataSource{}
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

func TestFlattenJobDataSource(t *testing.T) {
	t.Parallel()

	jobID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	description := "primary job"
	createdAt := time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	repoURL := "https://github.com/acme/web"

	data := flattenJobDataSource(JobDataSourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
		Slug:    types.StringValue("web"),
	}, sdk.AppJob{
		ID:               jobID,
		Slug:             "web",
		Name:             "Web",
		Description:      &description,
		ClusterID:        &clusterID,
		DefaultNamespace: "payments",
		RepoURL:          &repoURL,
		EffectiveRepoURL: repoURL,
		JobspecPath:      "jobs/web.nomad.hcl",
		JobspecType:      "hcl",
		Priority:         primaryJobPriority,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	})

	if data.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", data.OrgName.ValueString())
	}
	if data.AppSlug.ValueString() != "payments" {
		t.Fatalf("unexpected app_slug: %q", data.AppSlug.ValueString())
	}
	if data.Slug.ValueString() != "web" {
		t.Fatalf("unexpected slug: %q", data.Slug.ValueString())
	}
	if data.ID.ValueString() != jobID.String() {
		t.Fatalf("unexpected id: %q", data.ID.ValueString())
	}
	if data.Name.ValueString() != "Web" {
		t.Fatalf("unexpected name: %q", data.Name.ValueString())
	}
	if data.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", data.Description.ValueString())
	}
	if data.ClusterID.ValueString() != clusterID.String() {
		t.Fatalf("unexpected cluster_id: %q", data.ClusterID.ValueString())
	}
	if data.DefaultNamespace.ValueString() != "payments" {
		t.Fatalf("unexpected default_namespace: %q", data.DefaultNamespace.ValueString())
	}
	if data.RepoURL.ValueString() != repoURL {
		t.Fatalf("unexpected repo_url: %q", data.RepoURL.ValueString())
	}
	if data.EffectiveRepoURL.ValueString() != repoURL {
		t.Fatalf("unexpected effective_repo_url: %q", data.EffectiveRepoURL.ValueString())
	}
	if data.JobspecPath.ValueString() != "jobs/web.nomad.hcl" {
		t.Fatalf("unexpected jobspec_path: %q", data.JobspecPath.ValueString())
	}
	if data.JobspecType.ValueString() != "hcl" {
		t.Fatalf("unexpected jobspec_type: %q", data.JobspecType.ValueString())
	}
	if !data.IsPrimary.ValueBool() {
		t.Fatal("expected is_primary=true")
	}
	if data.Priority.ValueInt64() != int64(primaryJobPriority) {
		t.Fatalf("unexpected priority: %d", data.Priority.ValueInt64())
	}
	if data.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", data.CreatedAt.ValueString())
	}
	if data.UpdatedAt.ValueString() != updatedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", data.UpdatedAt.ValueString())
	}
}

func TestFlattenJobDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	jobID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))

	data := flattenJobDataSource(JobDataSourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
		Slug:    types.StringValue("web"),
	}, sdk.AppJob{
		ID:          jobID,
		Slug:        "web",
		Name:        "Web",
		JobspecPath: "jobs/web.nomad.hcl",
		JobspecType: "hcl",
		Priority:    primaryJobPriority * 2,
	})

	if !data.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !data.ClusterID.IsNull() {
		t.Fatal("expected cluster_id to be null")
	}
	if !data.RepoURL.IsNull() {
		t.Fatal("expected repo_url to be null")
	}
	if !data.EffectiveRepoURL.IsNull() {
		t.Fatal("expected effective_repo_url to be null")
	}
	if !data.DefaultNamespace.IsNull() {
		t.Fatal("expected default_namespace to be null")
	}
	if !data.CreatedAt.IsNull() {
		t.Fatal("expected created_at to be null")
	}
	if !data.UpdatedAt.IsNull() {
		t.Fatal("expected updated_at to be null")
	}
	if data.IsPrimary.ValueBool() {
		t.Fatal("expected is_primary=false")
	}
	if data.Priority.ValueInt64() != int64(primaryJobPriority*2) {
		t.Fatalf("unexpected priority: %d", data.Priority.ValueInt64())
	}
}

func assertJobDataSourceStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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

func assertJobDataSourceBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
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

func assertJobDataSourceInt64Attribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	intAttr, ok := attr.(schema.Int64Attribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.Int64Attribute, got %T", name, attr)
	}

	if intAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, intAttr.Required)
	}
	if intAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, intAttr.Optional)
	}
	if intAttr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, intAttr.Computed)
	}
}
