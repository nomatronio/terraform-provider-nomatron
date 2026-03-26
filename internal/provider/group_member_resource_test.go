package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewGroupMemberResource(t *testing.T) {
	t.Parallel()

	r := NewGroupMemberResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*GroupMemberResource); !ok {
		t.Fatalf("expected *GroupMemberResource, got %T", r)
	}
}

func TestGroupMemberResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &GroupMemberResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_group_member" {
		t.Fatalf("expected type name %q, got %q", "nomatron_group_member", resp.TypeName)
	}
}

func TestGroupMemberResource_Schema(t *testing.T) {
	t.Parallel()

	r := &GroupMemberResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "group_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "username", true, false, false, false)
}

func TestGroupMemberResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &GroupMemberResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestGroupMemberResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &GroupMemberResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestBuildGroupMemberID(t *testing.T) {
	t.Parallel()

	got := buildGroupMemberID("platform", "admins", "rbarnes")
	want := "group_name=admins&org_name=platform&username=rbarnes"

	if got != want {
		t.Fatalf("unexpected id: got %q want %q", got, want)
	}
}

func TestParseGroupMemberID(t *testing.T) {
	t.Parallel()

	orgName, groupName, username, err := parseGroupMemberID("group_name=admins&org_name=platform&username=rbarnes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgName != "platform" || groupName != "admins" || username != "rbarnes" {
		t.Fatalf("unexpected parsed values: %q %q %q", orgName, groupName, username)
	}
}

func TestParseGroupMemberID_Invalid(t *testing.T) {
	t.Parallel()

	if _, _, _, err := parseGroupMemberID("not-valid"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestGroupMemberNotFoundError(t *testing.T) {
	t.Parallel()

	err := &groupMemberNotFoundError{orgName: "platform", groupName: "admins", username: "rbarnes"}
	if !isGroupMemberNotFound(err) {
		t.Fatal("expected groupMemberNotFoundError to be recognized")
	}
	if isGroupMemberNotFound(errors.New("other")) {
		t.Fatal("expected non-group-member error to be ignored")
	}
}
