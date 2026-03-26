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

func TestNewOrganizationMemberResource(t *testing.T) {
	t.Parallel()

	r := NewOrganizationMemberResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*OrganizationMemberResource); !ok {
		t.Fatalf("expected *OrganizationMemberResource, got %T", r)
	}
}

func TestOrganizationMemberResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &OrganizationMemberResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_organization_member" {
		t.Fatalf("expected type name %q, got %q", "nomatron_organization_member", resp.TypeName)
	}
}

func TestOrganizationMemberResource_Schema(t *testing.T) {
	t.Parallel()

	r := &OrganizationMemberResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "username", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "user_id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "joined_at", false, false, true, false)
}

func TestOrganizationMemberResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &OrganizationMemberResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestOrganizationMemberResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &OrganizationMemberResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestBuildOrganizationMemberID(t *testing.T) {
	t.Parallel()

	got := buildOrganizationMemberID("platform", "rbarnes")
	want := "org_name=platform&username=rbarnes"

	if got != want {
		t.Fatalf("unexpected id: got %q want %q", got, want)
	}
}

func TestParseOrganizationMemberID(t *testing.T) {
	t.Parallel()

	orgName, username, err := parseOrganizationMemberID("org_name=platform&username=rbarnes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orgName != "platform" {
		t.Fatalf("unexpected org_name: %q", orgName)
	}
	if username != "rbarnes" {
		t.Fatalf("unexpected username: %q", username)
	}
}

func TestParseOrganizationMemberID_Invalid(t *testing.T) {
	t.Parallel()

	if _, _, err := parseOrganizationMemberID("not-valid"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestStateFromOrganizationMember(t *testing.T) {
	t.Parallel()

	userID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	joinedAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	base := OrganizationMemberResourceModel{
		OrgName:  types.StringValue("platform"),
		Username: types.StringValue("rbarnes"),
	}

	member := sdk.OrganizationMember{
		Username: "rbarnes",
		UserId:   userID,
		JoinedAt: joinedAt,
	}

	state := stateFromOrganizationMember(base, member)

	if state.ID.ValueString() != "org_name=platform&username=rbarnes" {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.UserID.ValueString() != userID.String() {
		t.Fatalf("unexpected user_id: %q", state.UserID.ValueString())
	}
	if state.JoinedAt.ValueString() != joinedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected joined_at: %q", state.JoinedAt.ValueString())
	}
}

func TestOrganizationMemberNotFoundError(t *testing.T) {
	t.Parallel()

	err := &organizationMemberNotFoundError{orgName: "platform", username: "rbarnes"}
	if !isOrganizationMemberNotFound(err) {
		t.Fatal("expected organizationMemberNotFoundError to be recognized")
	}
	if isOrganizationMemberNotFound(errors.New("other")) {
		t.Fatal("expected non-organization-member error to be ignored")
	}
}
