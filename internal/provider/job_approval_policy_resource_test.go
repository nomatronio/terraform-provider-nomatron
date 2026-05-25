package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNewJobApprovalPolicyResource(t *testing.T) {
	t.Parallel()

	r := NewJobApprovalPolicyResource()
	if r == nil {
		t.Fatal("expected resource to be non-nil")
	}

	if _, ok := r.(*JobApprovalPolicyResource); !ok {
		t.Fatalf("expected *JobApprovalPolicyResource, got %T", r)
	}
}

func TestJobApprovalPolicyResource_Metadata(t *testing.T) {
	t.Parallel()

	r := &JobApprovalPolicyResource{}
	req := resource.MetadataRequest{ProviderTypeName: "nomatron"}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "nomatron_approval_policy" {
		t.Fatalf("expected type name %q, got %q", "nomatron_approval_policy", resp.TypeName)
	}
}

func TestJobApprovalPolicyResource_Schema(t *testing.T) {
	t.Parallel()

	r := &JobApprovalPolicyResource{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attrs := resp.Schema.Attributes
	assertResourceStringAttribute(t, attrs, "id", false, false, true, false)
	assertResourceStringAttribute(t, attrs, "org_name", true, false, false, false)
	assertResourceStringAttribute(t, attrs, "app_slug", true, false, false, false)

	defaultRule, ok := attrs["default_rule"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected default_rule to be schema.SingleNestedAttribute, got %T", attrs["default_rule"])
	}
	if !defaultRule.Required {
		t.Fatal("expected default_rule to be required")
	}

	envRules, ok := attrs["environment_rules"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected environment_rules to be schema.ListNestedAttribute, got %T", attrs["environment_rules"])
	}
	if !envRules.Optional || !envRules.Computed {
		t.Fatal("expected environment_rules to be optional and computed")
	}
}

func TestBuildUpsertApprovalPolicyBody(t *testing.T) {
	t.Parallel()

	users, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"alice", "bob"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	groups, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"11111111-1111-1111-1111-111111111111"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	body, _, diags := buildUpsertApprovalPolicyBody(context.Background(), nil, JobApprovalPolicyResourceModel{
		DefaultRule: JobApprovalPolicyRuleModel{
			RequiredApprovals: types.Int64Value(2),
			Users:             users,
			Groups:            groups,
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if body.DefaultRule.RequiredApprovals != 2 {
		t.Fatalf("unexpected default required approvals: %d", body.DefaultRule.RequiredApprovals)
	}
	if len(body.DefaultRule.Approvers.Users) != 2 || body.DefaultRule.Approvers.Users[0] != "alice" {
		t.Fatalf("unexpected default users: %#v", body.DefaultRule.Approvers.Users)
	}
	if len(body.DefaultRule.Approvers.Groups) != 1 {
		t.Fatalf("unexpected default groups: %#v", body.DefaultRule.Approvers.Groups)
	}
	if len(body.EnvironmentRules) != 0 {
		t.Fatalf("unexpected environment rule count: %d", len(body.EnvironmentRules))
	}
}

func TestBuildUpsertApprovalPolicyBodyResolvesEnvironmentNames(t *testing.T) {
	t.Parallel()

	users, diags := types.ListValueFrom(context.Background(), types.StringType, []string{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	groups, diags := types.ListValueFrom(context.Background(), types.StringType, []string{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	envUsers, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"root"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	body, envIDsByName, diags := buildUpsertApprovalPolicyBodyWithEnvironmentNames(context.Background(), JobApprovalPolicyResourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
		DefaultRule: JobApprovalPolicyRuleModel{
			RequiredApprovals: types.Int64Value(0),
			Users:             users,
			Groups:            groups,
		},
		EnvironmentRules: []JobApprovalPolicyEnvironmentRuleModel{
			{
				EnvironmentID:     types.StringValue("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				RequiredApprovals: types.Int64Value(1),
				Users:             envUsers,
				Groups:            groups,
			},
		},
	}, map[string]string{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa": "dev",
	}, map[string]string{
		"dev": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(body.EnvironmentRules) != 1 {
		t.Fatalf("unexpected environment rule count: %d", len(body.EnvironmentRules))
	}
	if body.EnvironmentRules[0].Environment != "dev" {
		t.Fatalf("unexpected environment name: %q", body.EnvironmentRules[0].Environment)
	}
	if envIDsByName["dev"] != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected env id map: %#v", envIDsByName)
	}
}

func TestStateFromApprovalPolicy(t *testing.T) {
	t.Parallel()

	state, diags := stateFromApprovalPolicy(context.Background(), JobApprovalPolicyResourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
	}, approvalPolicyDTO{
		Version: 3,
		DefaultRule: approvalPolicyRuleDTO{
			RequiredApprovals: 2,
			Approvers: approvalPolicyApproversDTO{
				Users:  []string{"alice"},
				Groups: []string{"11111111-1111-1111-1111-111111111111"},
			},
		},
		EnvironmentRules: []approvalPolicyEnvironmentDTO{
			{
				Environment:       "dev",
				RequiredApprovals: 1,
				Approvers: approvalPolicyApproversDTO{
					Users:  []string{"bob"},
					Groups: []string{},
				},
			},
		},
	}, map[string]string{"dev": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ID.ValueString() != "platform/payments" {
		t.Fatalf("unexpected id: %q", state.ID.ValueString())
	}
	if state.Version.ValueInt64() != 3 {
		t.Fatalf("unexpected version: %d", state.Version.ValueInt64())
	}
	if state.DefaultRule.RequiredApprovals.ValueInt64() != 2 {
		t.Fatalf("unexpected default required approvals: %d", state.DefaultRule.RequiredApprovals.ValueInt64())
	}
	var users []string
	diags = state.DefaultRule.Users.ElementsAs(context.Background(), &users, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(users) != 1 || users[0] != "alice" {
		t.Fatalf("unexpected users: %#v", users)
	}
	if len(state.EnvironmentRules) != 1 {
		t.Fatalf("unexpected environment rule count: %d", len(state.EnvironmentRules))
	}
	if state.EnvironmentRules[0].EnvironmentID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected environment id: %q", state.EnvironmentRules[0].EnvironmentID.ValueString())
	}
}

func TestStateFromApprovalPolicyPreservesConfiguredApproverNames(t *testing.T) {
	t.Parallel()

	defaultUsers, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"root"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	defaultGroups, diags := types.ListValueFrom(context.Background(), types.StringType, []string{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	envUsers, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"root"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	envGroups, diags := types.ListValueFrom(context.Background(), types.StringType, []string{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	state, diags := stateFromApprovalPolicy(context.Background(), JobApprovalPolicyResourceModel{
		OrgName: types.StringValue("platform"),
		AppSlug: types.StringValue("payments"),
		DefaultRule: JobApprovalPolicyRuleModel{
			RequiredApprovals: types.Int64Value(0),
			Users:             defaultUsers,
			Groups:            defaultGroups,
		},
		EnvironmentRules: []JobApprovalPolicyEnvironmentRuleModel{
			{
				EnvironmentID:     types.StringValue("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				RequiredApprovals: types.Int64Value(1),
				Users:             envUsers,
				Groups:            envGroups,
			},
		},
	}, approvalPolicyDTO{
		Version: 2,
		DefaultRule: approvalPolicyRuleDTO{
			RequiredApprovals: 0,
			Approvers: approvalPolicyApproversDTO{
				Users:  []string{"50ef2390-9bc0-4007-a5e0-a29077cdc049"},
				Groups: []string{},
			},
		},
		EnvironmentRules: []approvalPolicyEnvironmentDTO{
			{
				Environment:       "prod",
				RequiredApprovals: 1,
				Approvers: approvalPolicyApproversDTO{
					Users:  []string{"50ef2390-9bc0-4007-a5e0-a29077cdc049"},
					Groups: []string{},
				},
			},
		},
	}, map[string]string{"prod": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var users []string
	diags = state.EnvironmentRules[0].Users.ElementsAs(context.Background(), &users, false)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(users) != 1 || users[0] != "root" {
		t.Fatalf("unexpected environment users: %#v", users)
	}
}

func TestParseApprovalPolicyImportID(t *testing.T) {
	t.Parallel()

	orgName, appSlug, err := parseApprovalPolicyImportID("platform/payments")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgName != "platform" || appSlug != "payments" {
		t.Fatalf("unexpected import id parts: %q %q", orgName, appSlug)
	}
}

func TestParseApprovalPolicyImportID_Invalid(t *testing.T) {
	t.Parallel()

	if _, _, err := parseApprovalPolicyImportID("platform/payments/web"); err == nil {
		t.Fatal("expected invalid import id to fail")
	}
}
