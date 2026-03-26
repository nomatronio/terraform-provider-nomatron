package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewVariableResource(t *testing.T) {
	t.Parallel()

	r := NewVariableResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*VariableResource); !ok {
		t.Fatalf("expected *VariableResource, got %T", r)
	}
}

func TestVariableResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &VariableResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_variable" {
		t.Fatalf("expected type name %q, got %q", "nomatron_variable", resp.TypeName)
	}
}

func TestVariableResource_Schema(t *testing.T) {
	t.Parallel()

	r := &VariableResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "scope", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "org_name", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "app_slug", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "job_slug", false, true, false, false)
	assertResourceStringAttribute(t, attrs, "key", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, true, false)
	assertResourceBoolAttribute(t, attrs, "sensitive", false, true, true)
	assertResourceStringAttribute(t, attrs, "value_type", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "value", false, true, true, false)
	assertResourceStringAttribute(t, attrs, "value_wo", false, true, false, true)
	assertResourceStringAttribute(t, attrs, "value_id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "updated_at", false, false, true, false)
}

func TestVariableResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &VariableResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestVariableResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &VariableResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestParseVariableImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw     string
		scope   string
		orgName string
		appSlug string
		jobSlug string
		id      string
	}{
		{"global/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "global", "", "", "", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
		{"organization/platform/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "organization", "platform", "", "", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
		{"app/platform/payments/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "app", "platform", "payments", "", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
		{"job/platform/payments/web/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "job", "platform", "payments", "web", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
	}

	for _, tt := range tests {
		scope, orgName, appSlug, jobSlug, id, err := parseVariableImportID(tt.raw)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.raw, err)
		}
		if scope != tt.scope || orgName != tt.orgName || appSlug != tt.appSlug || jobSlug != tt.jobSlug || id != tt.id {
			t.Fatalf("unexpected import parse result for %q", tt.raw)
		}
	}
}

func TestParseVariableImportID_Invalid(t *testing.T) {
	t.Parallel()

	if _, _, _, _, _, err := parseVariableImportID("job/platform/payments"); err == nil {
		t.Fatal("expected invalid import id to fail")
	}
}

func TestStateFromVariable_NonSensitive(t *testing.T) {
	t.Parallel()

	variableID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	valueID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 26, 18, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 26, 19, 0, 0, 0, time.UTC)
	description := "Application region"
	value := "us-west-2"

	state := stateFromVariable(VariableResourceModel{
		Scope:   types.StringValue("app"),
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
	}, sdk.Variable{
		Id:           variableID,
		Key:          "region",
		Description:  &description,
		Scope:        "app",
		Sensitivity:  "normal",
		ValueType:    "string",
		DefaultValue: &value,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, &sdk.VariableValue{
		Id:    valueID,
		Value: &value,
	})

	if state.ID.ValueString() != variableID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Scope.ValueString() != "app" {
		t.Fatalf("unexpected scope: %q", state.Scope.ValueString())
	}
	if state.Key.ValueString() != "region" {
		t.Fatalf("unexpected key: %q", state.Key.ValueString())
	}
	if state.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if state.Sensitive.ValueBool() {
		t.Fatal("expected sensitive=false")
	}
	if state.ValueType.ValueString() != "string" {
		t.Fatalf("unexpected value_type: %q", state.ValueType.ValueString())
	}
	if state.Value.ValueString() != value {
		t.Fatalf("unexpected value: %q", state.Value.ValueString())
	}
	if state.ValueID.ValueString() != valueID.String() {
		t.Fatalf("unexpected value_id: %q", state.ValueID.ValueString())
	}
}

func TestStateFromVariable_SensitiveMasksValue(t *testing.T) {
	t.Parallel()

	variableID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	valueID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))

	state := stateFromVariable(VariableResourceModel{
		Scope: types.StringValue("global"),
	}, sdk.Variable{
		Id:          variableID,
		Key:         "db_password",
		Scope:       "global",
		Sensitivity: "sensitive",
		ValueType:   "string",
	}, &sdk.VariableValue{
		Id: valueID,
	})

	if !state.Value.IsNull() {
		t.Fatal("expected sensitive value to be omitted from state")
	}
	if state.ValueID.ValueString() != valueID.String() {
		t.Fatalf("unexpected value_id: %q", state.ValueID.ValueString())
	}
	if !state.Sensitive.ValueBool() {
		t.Fatal("expected sensitive=true")
	}
}

func TestConfiguredVariableValue(t *testing.T) {
	t.Parallel()

	value, ok := configuredVariableValue(
		VariableResourceModel{
			Sensitive: types.BoolValue(false),
			Value:     types.StringValue("plain"),
		},
		VariableResourceModel{},
	)
	if !ok || value != "plain" {
		t.Fatal("expected non-sensitive value to be selected from value")
	}

	value, ok = configuredVariableValue(
		VariableResourceModel{
			Sensitive: types.BoolValue(true),
		},
		VariableResourceModel{
			ValueWO: types.StringValue("secret"),
		},
	)
	if !ok || value != "secret" {
		t.Fatal("expected sensitive value to be selected from value_wo")
	}
}
