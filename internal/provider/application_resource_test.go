package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewApplicationResource(t *testing.T) {
	t.Parallel()

	r := NewApplicationResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*ApplicationResource); !ok {
		t.Fatalf("expected *ApplicationResource, got %T", r)
	}
}

func TestApplicationResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &ApplicationResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_application" {
		t.Fatalf("expected type name %q, got %q", "nomatron_application", resp.TypeName)
	}
}

func TestApplicationResource_Schema(t *testing.T) {
	t.Parallel()

	r := &ApplicationResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "slug", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "cluster_id", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "repo_url", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "git_provider", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "ref", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "vcs_github_id", false, true, false, false)
	assertResourceBoolAttribute(t, attrs, "skip_repo_access_validation", false, true, false)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "updated_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "updated_by", false, false, true, false)
}

func TestApplicationResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &ApplicationResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestApplicationResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &ApplicationResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestStateFromApplication(t *testing.T) {
	t.Parallel()

	appID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	vcsGitHubID := openapi_types.UUID(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"))
	updatedBy := openapi_types.UUID(uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"))
	description := "primary app"
	ref := "main"
	createdAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 26, 13, 0, 0, 0, time.UTC)

	state := stateFromApplication(ApplicationResourceModel{
		OrgName:                  types.StringValue("platform"),
		SkipRepoAccessValidation: types.BoolValue(true),
	}, sdk.App{
		Id:          appID,
		Slug:        "payments",
		Name:        "Payments",
		Description: &description,
		ClusterId:   clusterID,
		RepoUrl:     "https://github.com/nomatron/payments",
		GitProvider: sdk.Github,
		Ref:         &ref,
		VcsGithubId: &vcsGitHubID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		UpdatedBy:   &updatedBy,
	})

	if state.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", state.OrgName.ValueString())
	}
	if state.ID.ValueString() != appID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Slug.ValueString() != "payments" {
		t.Fatalf("unexpected slug: %q", state.Slug.ValueString())
	}
	if state.Name.ValueString() != "Payments" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if state.ClusterID.ValueString() != clusterID.String() {
		t.Fatalf("unexpected cluster_id: %q", state.ClusterID.ValueString())
	}
	if state.RepoURL.ValueString() != "https://github.com/nomatron/payments" {
		t.Fatalf("unexpected repo_url: %q", state.RepoURL.ValueString())
	}
	if state.GitProvider.ValueString() != "github" {
		t.Fatalf("unexpected git_provider: %q", state.GitProvider.ValueString())
	}
	if state.Ref.ValueString() != ref {
		t.Fatalf("unexpected ref: %q", state.Ref.ValueString())
	}
	if state.VcsGitHubID.ValueString() != vcsGitHubID.String() {
		t.Fatalf("unexpected vcs_github_id: %q", state.VcsGitHubID.ValueString())
	}
	if !state.SkipRepoAccessValidation.ValueBool() {
		t.Fatal("expected skip_repo_access_validation to preserve configured value")
	}
	if state.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", state.CreatedAt.ValueString())
	}
	if state.UpdatedAt.ValueString() != updatedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected updated_at: %q", state.UpdatedAt.ValueString())
	}
	if state.UpdatedBy.ValueString() != updatedBy.String() {
		t.Fatalf("unexpected updated_by: %q", state.UpdatedBy.ValueString())
	}
}

func TestStateFromApplication_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	appID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))

	state := stateFromApplication(ApplicationResourceModel{
		OrgName:     types.StringValue("platform"),
		Description: types.StringNull(),
		Ref:         types.StringValue("main"),
		VcsGitHubID: types.StringNull(),
	}, sdk.App{
		Id:          appID,
		Slug:        "payments",
		Name:        "Payments",
		ClusterId:   clusterID,
		RepoUrl:     "https://github.com/nomatron/payments",
		GitProvider: sdk.Github,
	})

	if !state.Description.IsNull() {
		t.Fatal("expected description to be null")
	}
	if !state.VcsGitHubID.IsNull() {
		t.Fatal("expected vcs_github_id to be null")
	}
	if state.Ref.ValueString() != "main" {
		t.Fatalf("expected ref to preserve base state, got %q", state.Ref.ValueString())
	}
	if !state.CreatedAt.IsNull() {
		t.Fatal("expected created_at to be null")
	}
	if !state.UpdatedAt.IsNull() {
		t.Fatal("expected updated_at to be null")
	}
	if !state.UpdatedBy.IsNull() {
		t.Fatal("expected updated_by to be null")
	}
}

func TestStateFromApplication_PreservesConfiguredValuesWhenAPIOmitsFields(t *testing.T) {
	t.Parallel()

	appID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	clusterID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	state := stateFromApplication(ApplicationResourceModel{
		OrgName:     types.StringValue("platform"),
		ClusterID:   types.StringValue(clusterID),
		RepoURL:     types.StringValue("https://github.com/nomatron/payments"),
		GitProvider: types.StringValue("GitHub"),
	}, sdk.App{
		Id:   appID,
		Slug: "payments",
		Name: "Payments",
	})

	if state.ClusterID.ValueString() != clusterID {
		t.Fatalf("expected cluster_id to be preserved, got %q", state.ClusterID.ValueString())
	}
	if state.RepoURL.ValueString() != "https://github.com/nomatron/payments" {
		t.Fatalf("expected repo_url to be preserved, got %q", state.RepoURL.ValueString())
	}
	if state.GitProvider.ValueString() != "github" {
		t.Fatalf("expected git_provider to be preserved, got %q", state.GitProvider.ValueString())
	}
}

func TestLowerCaseStringPlanModifier(t *testing.T) {
	t.Parallel()

	resp := &planmodifier.StringResponse{}

	lowerCaseString().PlanModifyString(context.Background(), planmodifier.StringRequest{
		PlanValue: types.StringValue("GitHub"),
	}, resp)

	if resp.PlanValue.ValueString() != "github" {
		t.Fatalf("expected normalized plan value, got %q", resp.PlanValue.ValueString())
	}
}

func TestCreateApplicationParams_SkipRepoAccessValidation(t *testing.T) {
	t.Parallel()

	params := createApplicationParams(ApplicationResourceModel{
		SkipRepoAccessValidation: types.BoolValue(true),
	})
	if params == nil || params.Validate == nil {
		t.Fatal("expected create params with validate=false")
	}
	if *params.Validate {
		t.Fatal("expected validate=false when skip_repo_access_validation is true")
	}
}

func TestCreateApplicationParams_DefaultValidation(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]types.Bool{
		"unset":   types.BoolNull(),
		"unknown": types.BoolUnknown(),
		"false":   types.BoolValue(false),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			params := createApplicationParams(ApplicationResourceModel{
				SkipRepoAccessValidation: value,
			})
			if params != nil {
				t.Fatalf("expected nil params for %s, got %#v", name, params)
			}
		})
	}
}

func TestNormalizeProviderString(t *testing.T) {
	t.Parallel()

	if got := normalizeProviderString(" GitHub "); got != "github" {
		t.Fatalf("expected github, got %q", got)
	}
}

func TestParseApplicationImportID(t *testing.T) {
	t.Parallel()

	orgName, slug, err := parseApplicationImportID("platform/payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgName != "platform" || slug != "payments" {
		t.Fatalf("unexpected import id parts: %q %q", orgName, slug)
	}
}

func TestParseApplicationImportID_Invalid(t *testing.T) {
	t.Parallel()

	if _, _, err := parseApplicationImportID("platform"); err == nil {
		t.Fatal("expected invalid import id to fail")
	}
}

func TestApplicationNotFoundError(t *testing.T) {
	t.Parallel()

	err := &applicationNotFoundError{orgName: "platform", slug: "payments"}
	if !isApplicationNotFound(err) {
		t.Fatal("expected applicationNotFoundError to be recognized")
	}
	if isApplicationNotFound(errors.New("other")) {
		t.Fatal("expected non-application error to be ignored")
	}
}
