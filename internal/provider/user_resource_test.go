package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewUserResource(t *testing.T) {
	t.Parallel()

	r := NewUserResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*UserResource); !ok {
		t.Fatalf("expected *UserResource, got %T", r)
	}
}

func TestUserResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &UserResource{}
	req := resource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_user" {
		t.Fatalf("expected type name %q, got %q", "nomatron_user", resp.TypeName)
	}
}

func TestUserResource_Schema(t *testing.T) {
	t.Parallel()

	r := &UserResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes

	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "username", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "password", true, false, false, true)
	assertResourceMapAttribute(t, attrs, "metadata", false, true, false)
	assertResourceStringAttribute(t, attrs, "name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "auth_provider", false, false, true, false)
	assertResourceBoolAttribute(t, attrs, "is_active", false, false, true)
	assertResourceStringAttribute(t, attrs, "created_at", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "created_by", false, false, true, false)

	passwordAttr := attrs["password"].(schema.StringAttribute)
	if !passwordAttr.WriteOnly {
		t.Fatal("expected password to be write-only")
	}
	if !passwordAttr.Sensitive {
		t.Fatal("expected password to be sensitive")
	}

	usernameAttr := attrs["username"].(schema.StringAttribute)
	if len(usernameAttr.PlanModifiers) != 1 {
		t.Fatalf("expected username to have 1 plan modifier, got %d", len(usernameAttr.PlanModifiers))
	}

	metadataAttr := attrs["metadata"].(schema.MapAttribute)
	if len(metadataAttr.PlanModifiers) != 1 {
		t.Fatalf("expected metadata to have 1 plan modifier, got %d", len(metadataAttr.PlanModifiers))
	}
}

func TestUserResource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	r := &UserResource{}
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

func TestUserResource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	r := &UserResource{}
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

func TestUserResource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	r := &UserResource{}
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

func TestStateFromUser_WithMetadataAndUUIDCreatedBy(t *testing.T) {
	t.Parallel()

	userID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	creatorID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 25, 15, 4, 5, 0, time.UTC)

	var createdBy sdk.User_CreatedBy
	if err := createdBy.FromUserCreatedBy0(creatorID); err != nil {
		t.Fatalf("failed to build created_by union: %v", err)
	}

	md := map[string]string{
		"team":  "platform",
		"owner": "terraform",
	}

	base := UserResourceModel{
		Password: types.StringValue("super-secret"),
	}

	user := sdk.User{
		Id:           userID,
		Username:     "rbarnes",
		Name:         "Robert Barnes",
		AuthProvider: "nomatron",
		IsActive:     true,
		CreatedAt:    createdAt,
		CreatedBy:    createdBy,
		Metadata:     &md,
	}

	state, diags := stateFromUser(base, user)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Username.ValueString() != "rbarnes" {
		t.Fatalf("unexpected username: %q", state.Username.ValueString())
	}
	if state.Name.ValueString() != "Robert Barnes" {
		t.Fatalf("unexpected name: %q", state.Name.ValueString())
	}
	if state.Password.ValueString() != "super-secret" {
		t.Fatalf("expected password to be preserved from base, got %q", state.Password.ValueString())
	}
	if state.AuthProvider.ValueString() != "nomatron" {
		t.Fatalf("unexpected auth_provider: %q", state.AuthProvider.ValueString())
	}
	if !state.IsActive.ValueBool() {
		t.Fatal("expected is_active=true")
	}
	if state.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("unexpected created_at: %q", state.CreatedAt.ValueString())
	}
	if state.CreatedBy.ValueString() != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected created_by: %q", state.CreatedBy.ValueString())
	}
	if state.Metadata.IsNull() {
		t.Fatal("expected metadata to be populated")
	}

	gotMD := map[string]string{}
	diags = state.Metadata.ElementsAs(context.Background(), &gotMD, false)
	if diags.HasError() {
		t.Fatalf("unexpected metadata diagnostics: %v", diags)
	}
	if gotMD["team"] != "platform" || gotMD["owner"] != "terraform" {
		t.Fatalf("unexpected metadata: %#v", gotMD)
	}
}

func TestStateFromUser_WithNilMetadataAndStringCreatedBy(t *testing.T) {
	t.Parallel()

	userID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	createdAt := time.Date(2026, 3, 25, 15, 4, 5, 0, time.UTC)

	var createdBy sdk.User_CreatedBy
	if err := createdBy.FromUserCreatedBy1("bootstrap"); err != nil {
		t.Fatalf("failed to build created_by union: %v", err)
	}

	base := UserResourceModel{
		Password: types.StringNull(),
	}

	user := sdk.User{
		Id:           userID,
		Username:     "bootstrap",
		Name:         "Bootstrap User",
		AuthProvider: "nomatron",
		IsActive:     true,
		CreatedAt:    createdAt,
		CreatedBy:    createdBy,
		Metadata:     nil,
	}

	state, diags := stateFromUser(base, user)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !state.Metadata.IsNull() {
		t.Fatal("expected metadata to be null")
	}
	if state.CreatedBy.ValueString() != "bootstrap" {
		t.Fatalf("unexpected created_by: %q", state.CreatedBy.ValueString())
	}
}

func TestStateFromUser_PreservesBaseMetadataWhenResponseOmitsIt(t *testing.T) {
	t.Parallel()

	userID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	createdAt := time.Date(2026, 3, 25, 15, 4, 5, 0, time.UTC)

	var createdBy sdk.User_CreatedBy
	if err := createdBy.FromUserCreatedBy1("bootstrap"); err != nil {
		t.Fatalf("failed to build created_by union: %v", err)
	}

	baseMetadata, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"team":  "platform",
		"owner": "terraform",
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building metadata: %v", diags)
	}

	base := UserResourceModel{
		Metadata: baseMetadata,
		Password: types.StringNull(),
	}

	user := sdk.User{
		Id:           userID,
		Username:     "bootstrap",
		Name:         "Bootstrap User",
		AuthProvider: "nomatron",
		IsActive:     true,
		CreatedAt:    createdAt,
		CreatedBy:    createdBy,
		Metadata:     nil,
	}

	state, diags := stateFromUser(base, user)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	gotMD := map[string]string{}
	diags = state.Metadata.ElementsAs(context.Background(), &gotMD, false)
	if diags.HasError() {
		t.Fatalf("unexpected metadata diagnostics: %v", diags)
	}

	if gotMD["team"] != "platform" || gotMD["owner"] != "terraform" {
		t.Fatalf("expected base metadata to be preserved, got %#v", gotMD)
	}
}

func TestTerraformMapToStringMap_Null(t *testing.T) {
	t.Parallel()

	out, diags := terraformMapToStringMap(context.Background(), types.MapNull(types.StringType))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if out != nil {
		t.Fatalf("expected nil map, got %#v", out)
	}
}

func TestTerraformMapToStringMap_Value(t *testing.T) {
	t.Parallel()

	in, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"team": "platform",
		"env":  "dev",
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics building map: %v", diags)
	}

	out, diags := terraformMapToStringMap(context.Background(), in)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if out["team"] != "platform" || out["env"] != "dev" {
		t.Fatalf("unexpected output map: %#v", out)
	}
}

func TestMapStringToAny(t *testing.T) {
	t.Parallel()

	in := map[string]string{
		"team": "platform",
		"env":  "dev",
	}

	out := mapStringToAny(in)

	if out["team"] != "platform" {
		t.Fatalf("unexpected team value: %#v", out["team"])
	}
	if out["env"] != "dev" {
		t.Fatalf("unexpected env value: %#v", out["env"])
	}
}

func TestFormatAPIError_Nil(t *testing.T) {
	t.Parallel()

	got := formatAPIError(nil)
	if got != "unknown API error" {
		t.Fatalf("expected %q, got %q", "unknown API error", got)
	}
}

func TestFormatAPIError_NoErrors(t *testing.T) {
	t.Parallel()

	reqID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	apiErr := &sdk.ApiResponseError{
		Meta: sdk.ApiMeta{
			Code:      "VALIDATION_FAILED",
			Status:    sdk.ApiMetaStatus("error"),
			RequestId: reqID,
		},
		Errors: nil,
	}

	got := formatAPIError(apiErr)
	want := "VALIDATION_FAILED (error)"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatAPIError_WithField(t *testing.T) {
	t.Parallel()

	field := "username"
	apiErr := &sdk.ApiResponseError{
		Errors: []sdk.ApiErrorItem{
			{
				Code:    "VALIDATION_FAILED",
				Field:   &field,
				Message: "is required",
			},
		},
	}

	got := formatAPIError(apiErr)
	want := "username: is required"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatAPIError_WithoutField(t *testing.T) {
	t.Parallel()

	apiErr := &sdk.ApiResponseError{
		Errors: []sdk.ApiErrorItem{
			{
				Code:    "VALIDATION_FAILED",
				Message: "missing required fields",
			},
		},
	}

	got := formatAPIError(apiErr)
	want := "missing required fields"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func assertResourceStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed, sensitive bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	stringAttr, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.StringAttribute, got %T", name, attr)
	}

	if stringAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, stringAttr.Required)
	}
	if stringAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, stringAttr.Optional)
	}
	if stringAttr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, stringAttr.Computed)
	}
	if stringAttr.Sensitive != sensitive {
		t.Fatalf("expected attribute %q sensitive=%t, got %t", name, sensitive, stringAttr.Sensitive)
	}
}

func assertResourceBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	boolAttr, ok := attr.(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.BoolAttribute, got %T", name, attr)
	}

	if boolAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, boolAttr.Required)
	}
	if boolAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, boolAttr.Optional)
	}
	if boolAttr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, boolAttr.Computed)
	}
}

func assertResourceMapAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name]
	if !ok {
		t.Fatalf("expected attribute %q to exist", name)
	}

	mapAttr, ok := attr.(schema.MapAttribute)
	if !ok {
		t.Fatalf("expected attribute %q to be schema.MapAttribute, got %T", name, attr)
	}

	if mapAttr.Required != required {
		t.Fatalf("expected attribute %q required=%t, got %t", name, required, mapAttr.Required)
	}
	if mapAttr.Optional != optional {
		t.Fatalf("expected attribute %q optional=%t, got %t", name, optional, mapAttr.Optional)
	}
	if mapAttr.Computed != computed {
		t.Fatalf("expected attribute %q computed=%t, got %t", name, computed, mapAttr.Computed)
	}
}
