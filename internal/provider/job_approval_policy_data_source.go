package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &JobApprovalPolicyDataSource{}

type JobApprovalPolicyDataSource struct {
	client *sdk.ClientWithResponses
}

func NewJobApprovalPolicyDataSource() datasource.DataSource {
	return &JobApprovalPolicyDataSource{}
}

type JobApprovalPolicyDataSourceModel struct {
	ID               types.String                            `tfsdk:"id"`
	OrgName          types.String                            `tfsdk:"org_name"`
	AppSlug          types.String                            `tfsdk:"app_slug"`
	Version          types.Int64                             `tfsdk:"version"`
	DefaultRule      JobApprovalPolicyRuleModel              `tfsdk:"default_rule"`
	EnvironmentRules []JobApprovalPolicyEnvironmentRuleModel `tfsdk:"environment_rules"`
}

func (d *JobApprovalPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_approval_policy"
}

func (d *JobApprovalPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Fetch a Nomatron application approval policy by organization name and application slug.",
		Attributes: map[string]dschema.Attribute{
			"id": dschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable composite identifier for the approval policy.",
			},
			"org_name": dschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the application.",
			},
			"app_slug": dschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application slug that owns the approval policy.",
			},
			"version": dschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Approval policy version.",
			},
			"default_rule": dschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Default approval policy rule.",
				Attributes:          jobApprovalPolicyRuleDataSourceAttributes(),
			},
			"environment_rules": dschema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Environment-specific approval policy overrides.",
				NestedObject: dschema.NestedAttributeObject{
					Attributes: jobApprovalPolicyEnvironmentRuleDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *JobApprovalPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data JobApprovalPolicyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the approval policy data source.")
		return
	}

	base := JobApprovalPolicyResourceModel{
		OrgName: data.OrgName,
		AppSlug: data.AppSlug,
	}
	policy, err := readApprovalPolicy(ctx, d.client, data.OrgName.ValueString(), data.AppSlug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Approval Policy", err.Error())
		return
	}

	envIDsByName := map[string]string{}
	if len(policy.EnvironmentRules) > 0 {
		envIDsByName, err = approvalPolicyEnvironmentIDsByName(ctx, d.client, base)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Read Approval Policy Environments", err.Error())
			return
		}
	}

	state, diags := stateFromApprovalPolicy(ctx, base, policy, envIDsByName)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = JobApprovalPolicyDataSourceModel{
		ID:               state.ID,
		OrgName:          state.OrgName,
		AppSlug:          state.AppSlug,
		Version:          state.Version,
		DefaultRule:      state.DefaultRule,
		EnvironmentRules: state.EnvironmentRules,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *JobApprovalPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func jobApprovalPolicyRuleDataSourceAttributes() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"required_approvals": dschema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Number of approvals required by this rule.",
		},
		"users": dschema.ListAttribute{
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Usernames that can approve.",
		},
		"groups": dschema.ListAttribute{
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Group IDs that can approve.",
		},
	}
}

func jobApprovalPolicyEnvironmentRuleDataSourceAttributes() map[string]dschema.Attribute {
	attrs := jobApprovalPolicyRuleDataSourceAttributes()
	attrs["environment_id"] = dschema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Environment ID this rule applies to.",
	}
	return attrs
}
