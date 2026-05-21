package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

func TestNewDestroyJobAction(t *testing.T) {
	t.Parallel()

	a := NewDestroyJobAction()
	if a == nil {
		t.Fatal("expected action to be non-nil")
	}

	if _, ok := a.(*DestroyJobAction); !ok {
		t.Fatalf("expected *DestroyJobAction, got %T", a)
	}
}

func TestDestroyJobAction_Metadata(t *testing.T) {
	t.Parallel()

	a := &DestroyJobAction{}
	req := action.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &action.MetadataResponse{}

	a.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_destroy_job" {
		t.Fatalf("expected type name %q, got %q", "nomatron_destroy_job", resp.TypeName)
	}
}

func TestDestroyJobAction_Schema(t *testing.T) {
	t.Parallel()

	a := &DestroyJobAction{}
	resp := &action.SchemaResponse{}

	a.Schema(context.Background(), action.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes
	assertActionStringAttribute(t, attrs, "org_name", true, false)
	assertActionStringAttribute(t, attrs, "app_slug", true, false)
	assertActionStringAttribute(t, attrs, "job_slug", true, false)
	assertActionBoolAttribute(t, attrs, "apply", false, true)
}

func TestDestroyJobAction_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	a := &DestroyJobAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: nil}, resp)

	if a.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestDestroyJobAction_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	a := &DestroyJobAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if a.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestDestroyJobAction_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	a := &DestroyJobAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: client}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if a.client != client {
		t.Fatal("expected configured client to be stored on action")
	}
}
