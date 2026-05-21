package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

func TestNewDestroyAppJobsAction(t *testing.T) {
	t.Parallel()

	a := NewDestroyAppJobsAction()
	if a == nil {
		t.Fatal("expected action to be non-nil")
	}

	if _, ok := a.(*DestroyAppJobsAction); !ok {
		t.Fatalf("expected *DestroyAppJobsAction, got %T", a)
	}
}

func TestDestroyAppJobsAction_Metadata(t *testing.T) {
	t.Parallel()

	a := &DestroyAppJobsAction{}
	req := action.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &action.MetadataResponse{}

	a.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_destroy_app_jobs" {
		t.Fatalf("expected type name %q, got %q", "nomatron_destroy_app_jobs", resp.TypeName)
	}
}

func TestDestroyAppJobsAction_Schema(t *testing.T) {
	t.Parallel()

	a := &DestroyAppJobsAction{}
	resp := &action.SchemaResponse{}

	a.Schema(context.Background(), action.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes
	assertActionStringAttribute(t, attrs, "org_name", true, false)
	assertActionStringAttribute(t, attrs, "app_slug", true, false)
	assertActionBoolAttribute(t, attrs, "apply", false, true)
}

func TestDestroyAppJobsAction_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	a := &DestroyAppJobsAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: nil}, resp)

	if a.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestDestroyAppJobsAction_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	a := &DestroyAppJobsAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if a.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestDestroyAppJobsAction_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	a := &DestroyAppJobsAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: client}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if a.client != client {
		t.Fatal("expected configured client to be stored on action")
	}
}
