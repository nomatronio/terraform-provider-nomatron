package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

func TestNewImportNomadJobAction(t *testing.T) {
	t.Parallel()

	a := NewImportNomadJobAction()
	if a == nil {
		t.Fatal("expected action to be non-nil")
	}

	if _, ok := a.(*ImportNomadJobAction); !ok {
		t.Fatalf("expected *ImportNomadJobAction, got %T", a)
	}
}

func TestImportNomadJobAction_Metadata(t *testing.T) {
	t.Parallel()

	a := &ImportNomadJobAction{}
	req := action.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &action.MetadataResponse{}

	a.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_import_nomad_job" {
		t.Fatalf("expected type name %q, got %q", "nomatron_import_nomad_job", resp.TypeName)
	}
}

func TestImportNomadJobAction_Schema(t *testing.T) {
	t.Parallel()

	a := &ImportNomadJobAction{}
	resp := &action.SchemaResponse{}

	a.Schema(context.Background(), action.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertActionStringAttribute(t, attrs, "org_name", true, false)
	assertActionStringAttribute(t, attrs, "app_slug", true, false)
	assertActionStringAttribute(t, attrs, "job_slug", true, false)
	assertActionStringAttribute(t, attrs, "job_id", true, false)
}

func TestImportNomadJobAction_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	a := &ImportNomadJobAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: nil}, resp)

	if a.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestImportNomadJobAction_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	a := &ImportNomadJobAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if a.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestImportNomadJobAction_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	a := &ImportNomadJobAction{}
	resp := &action.ConfigureResponse{}

	a.Configure(context.Background(), action.ConfigureRequest{ProviderData: client}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if a.client != client {
		t.Fatal("expected configured client to be stored on action")
	}
}

func assertActionStringAttribute(t *testing.T, attrs map[string]actionschema.Attribute, name string, required, optional bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	stringAttr, ok := attr.(actionschema.StringAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be action schema.StringAttribute, got %T", name, attr)
	}

	if stringAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, stringAttr.Required)
	}
	if stringAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, stringAttr.Optional)
	}
}

func assertActionBoolAttribute(t *testing.T, attrs map[string]actionschema.Attribute, name string, required, optional bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	boolAttr, ok := attr.(actionschema.BoolAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be action schema.BoolAttribute, got %T", name, attr)
	}

	if boolAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, boolAttr.Required)
	}
	if boolAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, boolAttr.Optional)
	}
}
