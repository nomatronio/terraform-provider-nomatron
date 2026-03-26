package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

func TestNewGroupMemberDataSource(t *testing.T) {
	t.Parallel()

	ds := NewGroupMemberDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*GroupMemberDataSource); !ok {
		t.Fatalf("expected *GroupMemberDataSource, got %T", ds)
	}
}

func TestGroupMemberDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &GroupMemberDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "nomatron"}
	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	if resp.TypeName != "nomatron_group_member" {
		t.Fatalf("expected type name %q, got %q", "nomatron_group_member", resp.TypeName)
	}
}

func TestGroupMemberDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &GroupMemberDataSource{}
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes

	assertDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertDataSourceStringAttribute(t, attrs, "group_name", true, false, false)
	assertDataSourceStringAttribute(t, attrs, "username", true, false, false)
}

func TestGroupMemberDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &GroupMemberDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, &resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestGroupMemberDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &GroupMemberDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "not-a-client"}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestGroupMemberDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &GroupMemberDataSource{}
	var resp datasource.ConfigureResponse

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}
