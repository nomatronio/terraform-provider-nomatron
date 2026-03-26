package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewRoleAssignmentResource(t *testing.T) {
	t.Parallel()

	r := NewRoleAssignmentResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*RoleAssignmentResource); !ok {
		t.Fatalf("expected *RoleAssignmentResource, got %T", r)
	}
}

func TestRoleAssignmentResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &RoleAssignmentResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_role_assignment" {
		t.Fatalf("expected type name %q, got %q", "nomatron_role_assignment", resp.TypeName)
	}
}

func TestRoleAssignmentResource_Schema(t *testing.T) {
	t.Parallel()

	r := &RoleAssignmentResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "subject", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "role", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "domain", true, false, false, false)
}

func TestRoleAssignmentResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &RoleAssignmentResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestRoleAssignmentResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &RoleAssignmentResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestBuildRoleAssignmentID(t *testing.T) {
	t.Parallel()

	got := buildRoleAssignmentID("user:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "viewer", "global")
	want := "domain=global&role=viewer&subject=user%3Aaaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	if got != want {
		t.Fatalf("unexpected id: got %q want %q", got, want)
	}
}

func TestParseRoleAssignmentID(t *testing.T) {
	t.Parallel()

	subject, role, domain, err := parseRoleAssignmentID("domain=global&role=viewer&subject=user%3Aaaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if subject != "user:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	if role != "viewer" {
		t.Fatalf("unexpected role: %q", role)
	}
	if domain != "global" {
		t.Fatalf("unexpected domain: %q", domain)
	}
}

func TestParseRoleAssignmentID_Invalid(t *testing.T) {
	t.Parallel()

	if _, _, _, err := parseRoleAssignmentID("not-valid"); err == nil {
		t.Fatal("expected parse error")
	}
}
