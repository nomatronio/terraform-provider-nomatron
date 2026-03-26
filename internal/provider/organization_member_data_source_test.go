package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewOrganizationMemberDataSource(t *testing.T) {
	t.Parallel()

	ds := NewOrganizationMemberDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*OrganizationMemberDataSource); !ok {
		t.Fatalf("expected *OrganizationMemberDataSource, got %T", ds)
	}
}

func TestOrganizationMemberDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &OrganizationMemberDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "nomatron"}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_organization_member" {
		t.Fatalf("expected type name %q, got %q", "nomatron_organization_member", resp.TypeName)
	}
}

func TestOrganizationMemberDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &OrganizationMemberDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertDataSourceStringAttribute(t, attrs, "username", true, false, false)
	assertDataSourceStringAttribute(t, attrs, "user_id", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "joined_at", false, false, true)
}

func TestOrganizationMemberDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &OrganizationMemberDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, &resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestOrganizationMemberDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &OrganizationMemberDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "not-a-client"}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestOrganizationMemberDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &OrganizationMemberDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}

func TestFlattenOrganizationMemberDataSource(t *testing.T) {
	t.Parallel()

	userID := openapi_types.UUID(uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"))
	joinedAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	base := OrganizationMemberDataSourceModel{
		OrgName:  types.StringValue("platform"),
		Username: types.StringValue("rbarnes"),
	}

	member := sdk.OrganizationMember{
		Username: "rbarnes",
		UserId:   userID,
		JoinedAt: joinedAt,
	}

	data := flattenOrganizationMemberDataSource(base, member)

	if data.OrgName.ValueString() != "platform" {
		t.Fatalf("unexpected org_name: %q", data.OrgName.ValueString())
	}
	if data.Username.ValueString() != "rbarnes" {
		t.Fatalf("unexpected username: %q", data.Username.ValueString())
	}
	if data.UserID.ValueString() != userID.String() {
		t.Fatalf("unexpected user_id: %q", data.UserID.ValueString())
	}
	if data.JoinedAt.ValueString() != joinedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected joined_at: %q", data.JoinedAt.ValueString())
	}
}

func TestFlattenOrganizationMemberDataSource_WithNullOptionalFields(t *testing.T) {
	t.Parallel()

	base := OrganizationMemberDataSourceModel{
		OrgName:  types.StringValue("platform"),
		Username: types.StringValue("rbarnes"),
	}

	data := flattenOrganizationMemberDataSource(base, sdk.OrganizationMember{})

	if !data.UserID.IsNull() {
		t.Fatal("expected user_id to be null")
	}
	if !data.JoinedAt.IsNull() {
		t.Fatal("expected joined_at to be null")
	}
}
