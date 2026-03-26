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

func TestNewServiceAccountResource(t *testing.T) {
	t.Parallel()

	r := NewServiceAccountResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*ServiceAccountResource); !ok {
		t.Fatalf("expected *ServiceAccountResource, got %T", r)
	}
}

func TestServiceAccountResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &ServiceAccountResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_service_account" {
		t.Fatalf("expected type name %q, got %q", "nomatron_service_account", resp.TypeName)
	}
}

func TestServiceAccountResource_Schema(t *testing.T) {
	t.Parallel()

	r := &ServiceAccountResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "description", false, true, false, false)
	assertResourceBoolAttribute(t, attrs, "is_active", false, true, true)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "created_by", false, false, true, false)
}

func TestServiceAccountResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &ServiceAccountResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: nil}, resp)

	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestServiceAccountResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &ServiceAccountResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if r.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestServiceAccountResource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	r := &ServiceAccountResource{}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if r.client != client {
		t.Fatal("expected configured client to be stored on resource")
	}
}

func TestStateFromServiceAccount(t *testing.T) {
	t.Parallel()

	accountID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	creatorID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	description := "Automation service account"

	base := ServiceAccountResourceModel{
		Description: types.StringValue("old description"),
	}

	account := sdk.ServiceAccount{
		Id:          accountID,
		Name:        "terraform",
		Description: &description,
		IsActive:    true,
		CreatedAt:   createdAt,
		CreatedBy:   creatorID,
	}

	state := stateFromServiceAccount(base, account)

	if state.ID.ValueString() != accountID.String() {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "terraform" {
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
	if state.CreatedBy.ValueString() != creatorID.String() {
		t.Fatalf("unexpected created_by: %q", state.CreatedBy.ValueString())
	}
}

func TestStateFromServiceAccount_PreservesOptionalFieldsWhenMissing(t *testing.T) {
	t.Parallel()

	base := ServiceAccountResourceModel{
		ID:          types.StringValue("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name:        types.StringValue("terraform"),
		Description: types.StringValue("preserve me"),
		IsActive:    types.BoolValue(true),
		CreatedAt:   types.StringValue("2026-03-26T12:00:00Z"),
		CreatedBy:   types.StringValue("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
	}

	state := stateFromServiceAccount(base, sdk.ServiceAccount{})

	if state.ID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "terraform" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Description.ValueString() != "preserve me" {
		t.Fatalf("unexpected description: %q", state.Description.ValueString())
	}
	if !state.IsActive.ValueBool() {
		t.Fatal("expected is_active to be preserved")
	}
	if state.CreatedAt.ValueString() != "2026-03-26T12:00:00Z" {
		t.Fatalf("unexpected created_at: %q", state.CreatedAt.ValueString())
	}
	if state.CreatedBy.ValueString() != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected created_by: %q", state.CreatedBy.ValueString())
	}
}

func TestServiceAccountNotFoundError(t *testing.T) {
	t.Parallel()

	err := &serviceAccountNotFoundError{id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}
	if !isServiceAccountNotFound(err) {
		t.Fatal("expected serviceAccountNotFoundError to be recognized")
	}
	if isServiceAccountNotFound(errors.New("other")) {
		t.Fatal("expected non-service-account error to be ignored")
	}
}
