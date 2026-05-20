package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewJobResource(t *testing.T) {
	t.Parallel()

	r := NewJobResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*JobResource); !ok {
		t.Fatalf("expected *JobResource, got %T", r)
	}
}

func TestJobResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &JobResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_job" {
		t.Fatalf("expected type name %q, got %q", "nomatron_job", resp.TypeName)
	}
}

func TestJobResource_Schema(t *testing.T) {
	t.Parallel()

	r := &JobResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "app_slug", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "slug", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "cluster_id", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "default_namespace", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "repo_url", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "effective_repo_url", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "jobspec_path", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "jobspec_type", true, false, false, false)
	assertResourceBoolAttribute(t, attrs, "is_primary", false, true, true)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "updated_at", false, false, true, false)
}

func TestJobResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &JobResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestJobResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &JobResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestStateFromAppJob(t *testing.T) {
	t.Parallel()

	jobID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	description := "primary job"
	createdAt := time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 26, 15, 0, 0, 0, time.UTC)
	repoURL := "https://github.com/acme/web"

	state := stateFromAppJob(JobResourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
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

	if state.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", state.OrgName.ValueString())
	}
	if state.AppSlug.ValueString() != "payments" {
		t.Fatalf("unexpected app_slug: %q", state.AppSlug.ValueString())
	}
	if state.ID.ValueString() != jobID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Slug.ValueString() != "web" {
		t.Fatalf("unexpected slug: %q", state.Slug.ValueString())
	}
	if state.Name.ValueString() != "Web" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if state.ClusterID.ValueString() != clusterID.String() {
		t.Fatalf("unexpected cluster_id: %q", state.ClusterID.ValueString())
	}
	if state.DefaultNamespace.ValueString() != "payments" {
		t.Fatalf("unexpected default_namespace: %q", state.DefaultNamespace.ValueString())
	}
	if state.RepoURL.ValueString() != repoURL {
		t.Fatalf("unexpected repo_url: %q", state.RepoURL.ValueString())
	}
	if state.EffectiveRepoURL.ValueString() != repoURL {
		t.Fatalf("unexpected effective_repo_url: %q", state.EffectiveRepoURL.ValueString())
	}
	if state.JobspecPath.ValueString() != "jobs/web.nomad.hcl" {
		t.Fatalf("unexpected jobspec_path: %q", state.JobspecPath.ValueString())
	}
	if state.JobspecType.ValueString() != "hcl" {
		t.Fatalf("unexpected jobspec_type: %q", state.JobspecType.ValueString())
	}
	if !state.IsPrimary.ValueBool() {
		t.Fatal("expected is_primary=true")
	}
	if state.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", state.CreatedAt.ValueString())
	}
	if state.UpdatedAt.ValueString() != updatedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", state.UpdatedAt.ValueString())
	}
}

func TestStateFromAppJob_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	jobID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))

	state := stateFromAppJob(JobResourceModel{
		OrgName:          types.StringValue("platform"),
		AppSlug:          types.StringValue("payments"),
		Description:      types.StringNull(),
		ClusterID:        types.StringNull(),
		DefaultNamespace: types.StringValue("payments"),
	}, sdk.AppJob{
		ID:          jobID,
		Slug:        "web",
		Name:        "Web",
		JobspecPath: "jobs/web.nomad.hcl",
		JobspecType: "hcl",
		Priority:    primaryJobPriority * 2,
	})

	if !state.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !state.ClusterID.IsNull() {
		t.Fatal("expected cluster_id to be null")
	}
	if !state.RepoURL.IsNull() {
		t.Fatal("expected repo_url to be null")
	}
	if !state.EffectiveRepoURL.IsNull() {
		t.Fatal("expected effective_repo_url to be null")
	}
	if state.DefaultNamespace.ValueString() != "payments" {
		t.Fatalf("expected default_namespace to preserve base state, got %q", state.DefaultNamespace.ValueString())
	}
}

func TestBuildCreateAppJobBody_IncludesRepoOverride(t *testing.T) {
	t.Parallel()

	body, diags := buildCreateAppJobBody(JobResourceModel{
		Name:             types.StringValue("Web"),
		JobspecPath:      types.StringValue("jobs/web.nomad.hcl"),
		JobspecType:      types.StringValue("hcl"),
		DefaultNamespace: types.StringValue("payments"),
		RepoURL:          types.StringValue("https://github.com/acme/web"),
	})
	if diags.HasError() {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
	if body.RepoUrl == nil || *body.RepoUrl != "https://github.com/acme/web" {
		t.Fatalf("RepoUrl = %#v, want override URL", body.RepoUrl)
	}
}

func TestBuildUpdateAppJobBody_ClearsRepoOverride(t *testing.T) {
	t.Parallel()

	body, diags := buildUpdateAppJobBody(
		JobResourceModel{
			Name:        types.StringValue("Web"),
			JobspecPath: types.StringValue("jobs/web.nomad.hcl"),
			JobspecType: types.StringValue("hcl"),
			RepoURL:     types.StringNull(),
		},
		JobResourceModel{
			Name:        types.StringValue("Web"),
			JobspecPath: types.StringValue("jobs/web.nomad.hcl"),
			JobspecType: types.StringValue("hcl"),
			RepoURL:     types.StringValue("https://github.com/acme/web"),
		},
	)
	if diags.HasError() {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
	if body.RepoUrl == nil {
		t.Fatal("expected RepoUrl to be set for clear")
	}
	if *body.RepoUrl != "" {
		t.Fatalf("RepoUrl = %q, want empty string clear marker", *body.RepoUrl)
	}
}

func TestParseJobImportID(t *testing.T) {
	t.Parallel()

	orgName, appSlug, jobSlug, err := parseJobImportID("platform/payments/web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgName != "platform" || appSlug != "payments" || jobSlug != "web" {
		t.Fatalf("unexpected import id parts: %q %q %q", orgName, appSlug, jobSlug)
	}
}

func TestParseJobImportID_Invalid(t *testing.T) {
	t.Parallel()

	if _, _, _, err := parseJobImportID("platform/payments"); err == nil {
		t.Fatal("expected invalid import id to fail")
	}
}

func TestAppJobNotFoundError(t *testing.T) {
	t.Parallel()

	err := &appJobNotFoundError{orgName: "platform", appSlug: "payments", jobSlug: "web"}
	if !isAppJobNotFound(err) {
		t.Fatal("expected appJobNotFoundError to be recognized")
	}
	if isAppJobNotFound(errors.New("other")) {
		t.Fatal("expected non-job error to be ignored")
	}
}
