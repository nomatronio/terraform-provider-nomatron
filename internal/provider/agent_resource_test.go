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

func TestNewAgentResource(t *testing.T) {
	t.Parallel()

	r := NewAgentResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*AgentResource); !ok {
		t.Fatalf("expected *AgentResource, got %T", r)
	}
}

func TestAgentResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &AgentResource{}
	req := resource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_network_agent" {
		t.Fatalf("expected type name %q, got %q", "nomatron_network_agent", resp.TypeName)
	}
}

func TestAgentResource_Schema(t *testing.T) {
	t.Parallel()

	r := &AgentResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, false, false)
	assertResourceBoolAttribute(t, attrs, "is_active", false, true, true)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "created_by_type", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "created_by_id", false, false, true, false)
}

func TestAgentResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &AgentResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: nil,
	}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestAgentResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &AgentResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestAgentResource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	r := &AgentResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: client,
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if r.client != client {
		t.Fatal("expected configured client to be stored on resource")
	}
}

func TestStateFromAgent_WithCreatorAndLastSeen(t *testing.T) {
	t.Parallel()

	agentID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	userID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)
	description := "Private runner for edge workloads"

	base := AgentResourceModel{
		Description: types.StringValue("old description"),
	}

	agent := sdk.NetworkAgent{
		Id:              agentID,
		Name:            "edge-agent-1",
		Description:     &description,
		IsActive:        true,
		CreatedAt:       createdAt,
		CreatedByType:   sdk.NetworkAgentCreatedByTypeUser,
		CreatedByUserId: &userID,
	}

	state := stateFromAgent(base, agent)

	if state.ID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "edge-agent-1" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != description {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if !state.IsActive.ValueBool() {
		t.Fatal("expected is_active=true")
	}
	if state.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", state.CreatedAt.ValueString())
	}
	if state.CreatedByType.ValueString() != "user" {
		t.Fatalf("unexpected created_by_type: %q", state.CreatedByType.ValueString())
	}
	if state.CreatedByID.ValueString() != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected created_by_id: %q", state.CreatedByID.ValueString())
	}
}

func TestStateFromAgent_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	agentID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	serviceAccountID := openapi_types.UUID(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"))
	createdAt := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)
	base := AgentResourceModel{
		ID:          types.StringValue("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name:        types.StringValue("edge-agent-2"),
		Description: types.StringValue("preserve me"),
		CreatedAt:   types.StringValue(createdAt.Format(time.RFC3339)),
	}

	agent := sdk.NetworkAgent{
		Id:                        agentID,
		Name:                      "edge-agent-2",
		Description:               nil,
		IsActive:                  false,
		CreatedAt:                 createdAt,
		CreatedByType:             sdk.NetworkAgentCreatedByTypeServiceAccount,
		CreatedByServiceAccountId: &serviceAccountID,
	}

	state := stateFromAgent(base, agent)

	if state.ID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "edge-agent-2" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "preserve me" {
		t.Fatalf("expected description to preserve base value, got %q", state.Description.ValueString())
	}
	if state.CreatedByType.ValueString() != "service_account" {
		t.Fatalf("unexpected created_by_type: %q", state.CreatedByType.ValueString())
	}
	if state.CreatedByID.ValueString() != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Fatalf("unexpected created_by_id: %q", state.CreatedByID.ValueString())
	}
}

func TestStateFromAgent_PreservesBaseIdentityWhenResponseOmitsFields(t *testing.T) {
	t.Parallel()

	base := AgentResourceModel{
		ID:          types.StringValue("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name:        types.StringValue("test-agent"),
		Description: types.StringValue("a Terraform provisioned agent"),
		IsActive:    types.BoolValue(true),
		CreatedAt:   types.StringValue("2026-03-26T10:00:00Z"),
	}

	state := stateFromAgent(base, sdk.NetworkAgent{})

	if state.ID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected base id to be preserved, got %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "test-agent" {
		t.Fatalf("expected base name to be preserved, got %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "a Terraform provisioned agent" {
		t.Fatalf("expected base description to be preserved, got %q", state.Description.ValueString())
	}
}

func TestFlattenAgentCreatedByID(t *testing.T) {
	t.Parallel()

	userID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	serviceAccountID := openapi_types.UUID(uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"))

	gotUser := flattenAgentCreatedByID(sdk.NetworkAgent{
		CreatedByUserId: &userID,
	})
	if gotUser != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected user creator id: %q", gotUser)
	}

	gotServiceAccount := flattenAgentCreatedByID(sdk.NetworkAgent{
		CreatedByServiceAccountId: &serviceAccountID,
	})
	if gotServiceAccount != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Fatalf("unexpected service account creator id: %q", gotServiceAccount)
	}

	gotEmpty := flattenAgentCreatedByID(sdk.NetworkAgent{})
	if gotEmpty != "" {
		t.Fatalf("expected empty creator id, got %q", gotEmpty)
	}
}

func TestParseAgentID(t *testing.T) {
	t.Parallel()

	got, err := parseAgentID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.String() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected parsed UUID: %q", got.String())
	}
}

func TestParseAgentID_Invalid(t *testing.T) {
	t.Parallel()

	_, err := parseAgentID("not-a-uuid")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFormatErrorEnvelope_Nil(t *testing.T) {
	t.Parallel()

	got := formatErrorEnvelope(nil)
	if got != "unknown API error" {
		t.Fatalf("expected %q, got %q", "unknown API error", got)
	}
}

func TestFormatErrorEnvelope_NoErrors(t *testing.T) {
	t.Parallel()

	reqID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	apiErr := &sdk.ErrorEnvelope{
		Meta: sdk.ApiMeta{
			Code:      "VALIDATION_FAILED",
			Status:    sdk.ApiMetaStatus("error"),
			RequestId: reqID,
		},
	}

	got := formatErrorEnvelope(apiErr)
	want := "VALIDATION_FAILED (error)"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatErrorEnvelope_WithField(t *testing.T) {
	t.Parallel()

	field := "name"
	apiErr := &sdk.ErrorEnvelope{
		Errors: []sdk.ApiErrorItem{
			{
				Code:    "VALIDATION_FAILED",
				Field:   &field,
				Message: "is required",
			},
		},
	}

	got := formatErrorEnvelope(apiErr)
	want := "name: is required"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatErrorEnvelope_WithoutField(t *testing.T) {
	t.Parallel()

	apiErr := &sdk.ErrorEnvelope{
		Errors: []sdk.ApiErrorItem{
			{
				Code:    "VALIDATION_FAILED",
				Message: "invalid request",
			},
		},
	}

	got := formatErrorEnvelope(apiErr)
	want := "invalid request"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestStringValueChanged(t *testing.T) {
	t.Parallel()

	if stringValueChanged(types.StringNull(), types.StringNull()) {
		t.Fatal("expected null strings to be unchanged")
	}
	if !stringValueChanged(types.StringNull(), types.StringValue("x")) {
		t.Fatal("expected null/non-null strings to differ")
	}
	if stringValueChanged(types.StringUnknown(), types.StringValue("x")) {
		t.Fatal("expected unknown strings to avoid change detection")
	}
	if !stringValueChanged(types.StringValue("a"), types.StringValue("b")) {
		t.Fatal("expected differing strings to be detected")
	}
}

func TestBoolValueChanged(t *testing.T) {
	t.Parallel()

	if boolValueChanged(types.BoolNull(), types.BoolNull()) {
		t.Fatal("expected null bools to be unchanged")
	}
	if !boolValueChanged(types.BoolNull(), types.BoolValue(true)) {
		t.Fatal("expected null/non-null bools to differ")
	}
	if boolValueChanged(types.BoolUnknown(), types.BoolValue(true)) {
		t.Fatal("expected unknown bools to avoid change detection")
	}
	if !boolValueChanged(types.BoolValue(true), types.BoolValue(false)) {
		t.Fatal("expected differing bools to be detected")
	}
}
