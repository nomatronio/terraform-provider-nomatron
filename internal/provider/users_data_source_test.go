package provider

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestUsersDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &UsersDataSource{}
	req := datasource.MetadataRequest{
		ProviderTypeName: "nomatron",
	}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_users" {
		t.Fatalf("expected type name %q, got %q", "nomatron_users", resp.TypeName)
	}
}

func TestUsersDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &UsersDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	for _, name := range []string{"id", "username", "user", "users"} {
		if _, ok := attrs[name]; !ok {
			t.Fatalf("expected schema to contain attribute %q", name)
		}
	}
}

func TestUsersDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &UsersDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: nil,
	}, &resp)

	if ds.client != nil {
		t.Fatalf("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestUsersDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &UsersDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: &http.Client{},
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatalf("expected client to remain nil")
	}
}

func TestUsersDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to build sdk client: %v", err)
	}

	ds := &UsersDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{
		ProviderData: client,
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatalf("expected client to be configured")
	}
}

func TestFlattenCreatedBy_UUID(t *testing.T) {
	t.Parallel()

	id := openapi_types.UUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))

	var createdBy sdk.User_CreatedBy
	if err := createdBy.FromUserCreatedBy0(id); err != nil {
		t.Fatalf("failed to build created_by union: %v", err)
	}

	got := flattenCreatedBy(createdBy)
	want := "11111111-1111-1111-1111-111111111111"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFlattenCreatedBy_String(t *testing.T) {
	t.Parallel()

	var createdBy sdk.User_CreatedBy
	if err := createdBy.FromUserCreatedBy1("bootstrap"); err != nil {
		t.Fatalf("failed to build created_by union: %v", err)
	}

	got := flattenCreatedBy(createdBy)
	want := "bootstrap"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFlattenCreatedBy_EmptyOnInvalidUnion(t *testing.T) {
	t.Parallel()

	var createdBy sdk.User_CreatedBy

	got := flattenCreatedBy(createdBy)
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFlattenUser_WithMetadata(t *testing.T) {
	t.Parallel()

	userID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	creatorID := openapi_types.UUID(uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	createdAt := time.Date(2026, 3, 25, 10, 30, 0, 0, time.UTC)

	var createdBy sdk.User_CreatedBy
	if err := createdBy.FromUserCreatedBy0(creatorID); err != nil {
		t.Fatalf("failed to build created_by union: %v", err)
	}

	metadata := map[string]string{
		"team":  "platform",
		"owner": "rbarnes",
	}

	u := sdk.User{
		Id:           userID,
		Name:         "Robert Barnes",
		Username:     "rbarnes",
		IsActive:     true,
		AuthProvider: "local",
		CreatedAt:    createdAt,
		CreatedBy:    createdBy,
		Metadata:     &metadata,
	}

	got, diags := flattenUser(u)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	wantMetadata, mdDiags := types.MapValueFrom(context.Background(), types.StringType, metadata)
	if mdDiags.HasError() {
		t.Fatalf("unexpected metadata diagnostics: %v", mdDiags)
	}

	want, wantDiags := types.ObjectValue(
		userAttrTypes(),
		map[string]attr.Value{
			"id":            types.StringValue("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			"name":          types.StringValue("Robert Barnes"),
			"username":      types.StringValue("rbarnes"),
			"is_active":     types.BoolValue(true),
			"auth_provider": types.StringValue("local"),
			"created_at":    types.StringValue(createdAt.Format(time.RFC3339)),
			"created_by":    types.StringValue("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			"metadata":      wantMetadata,
		},
	)
	if wantDiags.HasError() {
		t.Fatalf("unexpected object diagnostics: %v", wantDiags)
	}

	if !got.Equal(want) {
		t.Fatalf("flattened user mismatch\nwant: %v\ngot:  %v", want, got)
	}
}

func TestFlattenUser_WithoutMetadata(t *testing.T) {
	t.Parallel()

	userID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	createdAt := time.Date(2026, 3, 25, 10, 30, 0, 0, time.UTC)

	var createdBy sdk.User_CreatedBy
	if err := createdBy.FromUserCreatedBy1("bootstrap"); err != nil {
		t.Fatalf("failed to build created_by union: %v", err)
	}

	u := sdk.User{
		Id:           userID,
		Name:         "Bootstrap User",
		Username:     "bootstrap",
		IsActive:     true,
		AuthProvider: "local",
		CreatedAt:    createdAt,
		CreatedBy:    createdBy,
		Metadata:     nil,
	}

	got, diags := flattenUser(u)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want, wantDiags := types.ObjectValue(
		userAttrTypes(),
		map[string]attr.Value{
			"id":            types.StringValue("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			"name":          types.StringValue("Bootstrap User"),
			"username":      types.StringValue("bootstrap"),
			"is_active":     types.BoolValue(true),
			"auth_provider": types.StringValue("local"),
			"created_at":    types.StringValue(createdAt.Format(time.RFC3339)),
			"created_by":    types.StringValue("bootstrap"),
			"metadata":      types.MapNull(types.StringType),
		},
	)
	if wantDiags.HasError() {
		t.Fatalf("unexpected object diagnostics: %v", wantDiags)
	}

	if !got.Equal(want) {
		t.Fatalf("flattened user mismatch\nwant: %v\ngot:  %v", want, got)
	}
}
