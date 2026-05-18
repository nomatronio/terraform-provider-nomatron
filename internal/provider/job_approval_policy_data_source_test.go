package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

func TestNewJobApprovalPolicyDataSource(t *testing.T) {
	t.Parallel()

	ds := NewJobApprovalPolicyDataSource()
	if ds == nil {
		t.Fatal("expected data source to be non-nil")
	}

	if _, ok := ds.(*JobApprovalPolicyDataSource); !ok {
		t.Fatalf("expected *JobApprovalPolicyDataSource, got %T", ds)
	}
}

func TestJobApprovalPolicyDataSource_Metadata(t *testing.T) {
	t.Parallel()

	ds := &JobApprovalPolicyDataSource{}
	req := datasource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_approval_policy" {
		t.Fatalf("expected type name %q, got %q", "nomatron_approval_policy", resp.TypeName)
	}
}

func TestJobApprovalPolicyDataSource_Schema(t *testing.T) {
	t.Parallel()

	ds := &JobApprovalPolicyDataSource{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes
	assertDataSourceStringAttribute(t, attrs, "id", false, false, true)
	assertDataSourceStringAttribute(t, attrs, "org_name", true, false, false)
	assertDataSourceStringAttribute(t, attrs, "app_slug", true, false, false)

	defaultRule, ok := attrs["default_rule"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected default_rule to be schema.SingleNestedAttribute, got %T", attrs["default_rule"])
	}
	if !defaultRule.Computed {
		t.Fatal("expected default_rule to be computed")
	}

	envRules, ok := attrs["environment_rules"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected environment_rules to be schema.ListNestedAttribute, got %T", attrs["environment_rules"])
	}
	if !envRules.Computed {
		t.Fatal("expected environment_rules to be computed")
	}
}

func TestJobApprovalPolicyDataSource_ConfigureNilProviderData(t *testing.T) {
	t.Parallel()

	ds := &JobApprovalPolicyDataSource{}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: nil}, resp)

	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestJobApprovalPolicyDataSource_ConfigureWrongType(t *testing.T) {
	t.Parallel()

	ds := &JobApprovalPolicyDataSource{}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "not-a-client"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics error for wrong provider data type")
	}
	if ds.client != nil {
		t.Fatal("expected client to remain nil")
	}
}

func TestJobApprovalPolicyDataSource_ConfigureSetsClient(t *testing.T) {
	t.Parallel()

	client, err := sdk.NewClientWithResponses("http://localhost:4649/api/v1")
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	ds := &JobApprovalPolicyDataSource{}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: client}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
	if ds.client != client {
		t.Fatal("expected configured client to be stored on data source")
	}
}
