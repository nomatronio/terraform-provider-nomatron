package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &JobApprovalPolicyResource{}
var _ resource.ResourceWithImportState = &JobApprovalPolicyResource{}

type JobApprovalPolicyResource struct {
	client *sdk.ClientWithResponses
}

func NewJobApprovalPolicyResource() resource.Resource {
	return &JobApprovalPolicyResource{}
}

type JobApprovalPolicyResourceModel struct {
	ID               types.String                            `tfsdk:"id"`
	OrgName          types.String                            `tfsdk:"org_name"`
	AppSlug          types.String                            `tfsdk:"app_slug"`
	JobSlug          types.String                            `tfsdk:"job_slug"`
	Version          types.Int64                             `tfsdk:"version"`
	DefaultRule      JobApprovalPolicyRuleModel              `tfsdk:"default_rule"`
	EnvironmentRules []JobApprovalPolicyEnvironmentRuleModel `tfsdk:"environment_rules"`
}

type JobApprovalPolicyRuleModel struct {
	RequiredApprovals types.Int64 `tfsdk:"required_approvals"`
	Users             types.List  `tfsdk:"users"`
	Groups            types.List  `tfsdk:"groups"`
}

type JobApprovalPolicyEnvironmentRuleModel struct {
	EnvironmentID     types.String `tfsdk:"environment_id"`
	RequiredApprovals types.Int64  `tfsdk:"required_approvals"`
	Users             types.List   `tfsdk:"users"`
	Groups            types.List   `tfsdk:"groups"`
}

func (r *JobApprovalPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job_approval_policy"
}

func (r *JobApprovalPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron job approval policy resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable composite identifier for the approval policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application slug that owns the job.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"job_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Job slug that owns the approval policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Approval policy version.",
			},
			"default_rule": jobApprovalPolicyRuleResourceSchema(true),
			"environment_rules": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Environment-specific approval policy overrides.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: jobApprovalPolicyEnvironmentRuleResourceAttributes(),
				},
			},
		},
	}
}

func (r *JobApprovalPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *JobApprovalPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan JobApprovalPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the job approval policy resource.")
		return
	}

	body, diags := buildUpsertApprovalPolicyBody(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := upsertApprovalPolicy(ctx, r.client, plan.OrgName.ValueString(), plan.AppSlug.ValueString(), plan.JobSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Upsert Job Approval Policy", err.Error())
		return
	}

	state, diags := stateFromApprovalPolicy(ctx, plan, policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *JobApprovalPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state JobApprovalPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the job approval policy resource.")
		return
	}

	policy, err := readApprovalPolicy(ctx, r.client, state.OrgName.ValueString(), state.AppSlug.ValueString(), state.JobSlug.ValueString())
	if err != nil {
		if isApprovalPolicyNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Job Approval Policy", err.Error())
		return
	}

	newState, diags := stateFromApprovalPolicy(ctx, state, policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *JobApprovalPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan JobApprovalPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the job approval policy resource.")
		return
	}

	body, diags := buildUpsertApprovalPolicyBody(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := upsertApprovalPolicy(ctx, r.client, plan.OrgName.ValueString(), plan.AppSlug.ValueString(), plan.JobSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Job Approval Policy", err.Error())
		return
	}

	state, diags := stateFromApprovalPolicy(ctx, plan, policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *JobApprovalPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state JobApprovalPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the job approval policy resource.")
		return
	}

	body := sdk.UpsertApprovalPolicyJSONRequestBody{
		DefaultRule: sdk.UpsertApprovalRule{
			RequiredApprovals: 0,
			Approvers: sdk.UpsertApprovalApprovers{
				Users:  []string{},
				Groups: []string{},
			},
		},
		EnvironmentRules: []sdk.UpsertEnvironmentApprovalRule{},
	}

	if _, err := upsertApprovalPolicy(ctx, r.client, state.OrgName.ValueString(), state.AppSlug.ValueString(), state.JobSlug.ValueString(), body); err != nil {
		if isApprovalPolicyNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed To Clear Job Approval Policy", err.Error())
		return
	}
}

func (r *JobApprovalPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgName, appSlug, jobSlug, err := parseJobImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_slug"), appSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("job_slug"), jobSlug)...)
}

type approvalPolicyNotFoundError struct {
	orgName string
	appSlug string
	jobSlug string
}

func (e *approvalPolicyNotFoundError) Error() string {
	return fmt.Sprintf("approval policy for job %q in app %q in org %q not found", e.jobSlug, e.appSlug, e.orgName)
}

func isApprovalPolicyNotFound(err error) bool {
	_, ok := err.(*approvalPolicyNotFoundError)
	return ok
}

func readApprovalPolicy(ctx context.Context, client *sdk.ClientWithResponses, orgName, appSlug, jobSlug string) (sdk.ApprovalPolicy, error) {
	rsp, err := client.GetApprovalPolicyWithResponse(ctx, orgName, appSlug, jobSlug)
	if err != nil {
		return sdk.ApprovalPolicy{}, err
	}
	if rsp.JSON404 != nil {
		return sdk.ApprovalPolicy{}, &approvalPolicyNotFoundError{orgName: orgName, appSlug: appSlug, jobSlug: jobSlug}
	}
	if rsp.JSON401 != nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("unauthorized reading approval policy: %s", formatAPIError(rsp.JSON401))
	}
	if rsp.JSON403 != nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("forbidden reading approval policy: %s", formatAPIError(rsp.JSON403))
	}
	if rsp.JSON500 != nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("failed to read approval policy: %s", formatAPIError(rsp.JSON500))
	}
	if rsp.JSON200 == nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("expected 200 response when reading approval policy for job %q in app %q in org %q, got %s", jobSlug, appSlug, orgName, rsp.Status())
	}
	return rsp.JSON200.Data, nil
}

func upsertApprovalPolicy(ctx context.Context, client *sdk.ClientWithResponses, orgName, appSlug, jobSlug string, body sdk.UpsertApprovalPolicyJSONRequestBody) (sdk.ApprovalPolicy, error) {
	rsp, err := client.UpsertApprovalPolicyWithResponse(ctx, orgName, appSlug, jobSlug, body)
	if err != nil {
		return sdk.ApprovalPolicy{}, err
	}
	if rsp.JSON404 != nil {
		return sdk.ApprovalPolicy{}, &approvalPolicyNotFoundError{orgName: orgName, appSlug: appSlug, jobSlug: jobSlug}
	}
	if rsp.JSON400 != nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("failed to upsert approval policy: %s", formatAPIError(rsp.JSON400))
	}
	if rsp.JSON401 != nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("unauthorized upserting approval policy: %s", formatAPIError(rsp.JSON401))
	}
	if rsp.JSON403 != nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("forbidden upserting approval policy: %s", formatAPIError(rsp.JSON403))
	}
	if rsp.JSON500 != nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("failed to upsert approval policy: %s", formatAPIError(rsp.JSON500))
	}
	if rsp.JSON200 == nil {
		return sdk.ApprovalPolicy{}, fmt.Errorf("expected 200 response when upserting approval policy for job %q in app %q in org %q, got %s", jobSlug, appSlug, orgName, rsp.Status())
	}
	return rsp.JSON200.Data, nil
}

func buildUpsertApprovalPolicyBody(ctx context.Context, plan JobApprovalPolicyResourceModel) (sdk.UpsertApprovalPolicyJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	defaultRule, ruleDiags := buildUpsertApprovalRule(ctx, plan.DefaultRule)
	diags.Append(ruleDiags...)
	if diags.HasError() {
		return sdk.UpsertApprovalPolicyJSONRequestBody{}, diags
	}

	envRules := make([]sdk.UpsertEnvironmentApprovalRule, 0, len(plan.EnvironmentRules))
	for _, rule := range plan.EnvironmentRules {
		envID, err := parseApplicationID(rule.EnvironmentID.ValueString())
		if err != nil {
			diags.AddError("Invalid Environment ID", err.Error())
			return sdk.UpsertApprovalPolicyJSONRequestBody{}, diags
		}
		upsertRule, ruleDiags := buildUpsertApprovalRule(ctx, JobApprovalPolicyRuleModel{
			RequiredApprovals: rule.RequiredApprovals,
			Users:             rule.Users,
			Groups:            rule.Groups,
		})
		diags.Append(ruleDiags...)
		if diags.HasError() {
			return sdk.UpsertApprovalPolicyJSONRequestBody{}, diags
		}
		envRules = append(envRules, sdk.UpsertEnvironmentApprovalRule{
			EnvironmentId: envID,
			Rule:          upsertRule,
		})
	}

	return sdk.UpsertApprovalPolicyJSONRequestBody{
		DefaultRule:      defaultRule,
		EnvironmentRules: envRules,
	}, diags
}

func buildUpsertApprovalRule(ctx context.Context, rule JobApprovalPolicyRuleModel) (sdk.UpsertApprovalRule, diag.Diagnostics) {
	var diags diag.Diagnostics

	users, userDiags := terraformListToStringSlice(ctx, rule.Users)
	diags.Append(userDiags...)
	groups, groupDiags := terraformListToStringSlice(ctx, rule.Groups)
	diags.Append(groupDiags...)
	if diags.HasError() {
		return sdk.UpsertApprovalRule{}, diags
	}
	if users == nil {
		users = []string{}
	}
	if groups == nil {
		groups = []string{}
	}

	return sdk.UpsertApprovalRule{
		RequiredApprovals: int(rule.RequiredApprovals.ValueInt64()),
		Approvers: sdk.UpsertApprovalApprovers{
			Users:  users,
			Groups: groups,
		},
	}, diags
}

func stateFromApprovalPolicy(ctx context.Context, base JobApprovalPolicyResourceModel, policy sdk.ApprovalPolicy) (JobApprovalPolicyResourceModel, diag.Diagnostics) {
	defaultRule, diags := stateFromApprovalRule(ctx, policy.DefaultRule)
	if diags.HasError() {
		return JobApprovalPolicyResourceModel{}, diags
	}

	envRules := make([]JobApprovalPolicyEnvironmentRuleModel, 0, len(policy.EnvironmentRules))
	for _, rule := range policy.EnvironmentRules {
		ruleState, ruleDiags := stateFromApprovalRule(ctx, rule.Rule)
		diags.Append(ruleDiags...)
		if diags.HasError() {
			return JobApprovalPolicyResourceModel{}, diags
		}
		envRules = append(envRules, JobApprovalPolicyEnvironmentRuleModel{
			EnvironmentID:     types.StringValue(rule.EnvironmentId.String()),
			RequiredApprovals: ruleState.RequiredApprovals,
			Users:             ruleState.Users,
			Groups:            ruleState.Groups,
		})
	}

	return JobApprovalPolicyResourceModel{
		ID:               types.StringValue(approvalPolicyID(base.OrgName.ValueString(), base.AppSlug.ValueString(), base.JobSlug.ValueString())),
		OrgName:          base.OrgName,
		AppSlug:          base.AppSlug,
		JobSlug:          base.JobSlug,
		Version:          types.Int64Value(int64(policy.Version)),
		DefaultRule:      defaultRule,
		EnvironmentRules: envRules,
	}, diags
}

func stateFromApprovalRule(ctx context.Context, rule sdk.ApprovalRule) (JobApprovalPolicyRuleModel, diag.Diagnostics) {
	users, diags := types.ListValueFrom(ctx, types.StringType, rule.Approvers.Users)
	if diags.HasError() {
		return JobApprovalPolicyRuleModel{}, diags
	}

	groups := make([]string, 0, len(rule.Approvers.Groups))
	for _, group := range rule.Approvers.Groups {
		groups = append(groups, group.String())
	}
	groupValues, groupDiags := types.ListValueFrom(ctx, types.StringType, groups)
	diags.Append(groupDiags...)
	if diags.HasError() {
		return JobApprovalPolicyRuleModel{}, diags
	}

	return JobApprovalPolicyRuleModel{
		RequiredApprovals: types.Int64Value(int64(rule.RequiredApprovals)),
		Users:             users,
		Groups:            groupValues,
	}, diags
}

func approvalPolicyID(orgName, appSlug, jobSlug string) string {
	return strings.Join([]string{orgName, appSlug, jobSlug}, "/")
}

func jobApprovalPolicyRuleResourceSchema(required bool) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Required:            required,
		MarkdownDescription: "Approval policy rule.",
		Attributes:          jobApprovalPolicyRuleResourceAttributes(),
	}
}

func jobApprovalPolicyRuleResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"required_approvals": schema.Int64Attribute{
			Required:            true,
			MarkdownDescription: "Number of approvals required by this rule.",
		},
		"users": schema.ListAttribute{
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Usernames that can approve.",
		},
		"groups": schema.ListAttribute{
			Optional:            true,
			Computed:            true,
			ElementType:         types.StringType,
			MarkdownDescription: "Group IDs that can approve.",
		},
	}
}

func jobApprovalPolicyEnvironmentRuleResourceAttributes() map[string]schema.Attribute {
	attrs := jobApprovalPolicyRuleResourceAttributes()
	attrs["environment_id"] = schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Environment ID this rule applies to.",
	}
	return attrs
}
