package provider

import (
	"bytes"
	"context"
	"encoding/json"
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

type approvalPolicyDTO struct {
	Version          int                            `json:"version"`
	DefaultRule      approvalPolicyRuleDTO          `json:"default_rule"`
	EnvironmentRules []approvalPolicyEnvironmentDTO `json:"environment_rules,omitempty"`
}

type approvalPolicyRuleDTO struct {
	RequiredApprovals int                        `json:"required_approvals"`
	Approvers         approvalPolicyApproversDTO `json:"approvers"`
}

type approvalPolicyEnvironmentDTO struct {
	Environment       string                     `json:"environment"`
	RequiredApprovals int                        `json:"required_approvals"`
	Approvers         approvalPolicyApproversDTO `json:"approvers"`
}

type approvalPolicyApproversDTO struct {
	Users  []string `json:"users"`
	Groups []string `json:"groups"`
}

type approvalPolicyAPIResponse struct {
	Data approvalPolicyDTO `json:"data"`
}

func (r *JobApprovalPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_approval_policy"
}

func (r *JobApprovalPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron application approval policy resource.",
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
				MarkdownDescription: "Application slug that owns the approval policy.",
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
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the approval policy resource.")
		return
	}

	body, envIDsByName, diags := buildUpsertApprovalPolicyBody(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := upsertApprovalPolicy(ctx, r.client, plan.OrgName.ValueString(), plan.AppSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Upsert Approval Policy", err.Error())
		return
	}

	state, diags := stateFromApprovalPolicy(ctx, plan, policy, envIDsByName)
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
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the approval policy resource.")
		return
	}

	policy, err := readApprovalPolicy(ctx, r.client, state.OrgName.ValueString(), state.AppSlug.ValueString())
	if err != nil {
		if isApprovalPolicyNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Approval Policy", err.Error())
		return
	}

	envIDsByName := map[string]string{}
	if len(policy.EnvironmentRules) > 0 {
		envIDsByName, err = approvalPolicyEnvironmentIDsByName(ctx, r.client, state)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Read Approval Policy Environments", err.Error())
			return
		}
	}

	newState, diags := stateFromApprovalPolicy(ctx, state, policy, envIDsByName)
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
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the approval policy resource.")
		return
	}

	body, envIDsByName, diags := buildUpsertApprovalPolicyBody(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := upsertApprovalPolicy(ctx, r.client, plan.OrgName.ValueString(), plan.AppSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Approval Policy", err.Error())
		return
	}

	state, diags := stateFromApprovalPolicy(ctx, plan, policy, envIDsByName)
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
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the approval policy resource.")
		return
	}

	body := approvalPolicyDTO{
		DefaultRule: approvalPolicyRuleDTO{
			RequiredApprovals: 0,
			Approvers: approvalPolicyApproversDTO{
				Users:  []string{},
				Groups: []string{},
			},
		},
		EnvironmentRules: []approvalPolicyEnvironmentDTO{},
	}

	if _, err := upsertApprovalPolicy(ctx, r.client, state.OrgName.ValueString(), state.AppSlug.ValueString(), body); err != nil {
		if isApprovalPolicyNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed To Clear Approval Policy", err.Error())
		return
	}
}

func (r *JobApprovalPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgName, appSlug, err := parseApprovalPolicyImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_slug"), appSlug)...)
}

type approvalPolicyNotFoundError struct {
	orgName string
	appSlug string
}

func (e *approvalPolicyNotFoundError) Error() string {
	return fmt.Sprintf("approval policy for app %q in org %q not found", e.appSlug, e.orgName)
}

func isApprovalPolicyNotFound(err error) bool {
	_, ok := err.(*approvalPolicyNotFoundError)
	return ok
}

func parseApprovalPolicyImportID(raw string) (orgName, appSlug string, err error) {
	parts := strings.Split(raw, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected import identifier in the format `org_name/app_slug`")
	}
	return parts[0], parts[1], nil
}

func readApprovalPolicy(ctx context.Context, client *sdk.ClientWithResponses, orgName, appSlug string) (approvalPolicyDTO, error) {
	rsp, err := client.GetApprovalPolicyWithResponse(ctx, orgName, appSlug)
	if err != nil {
		return approvalPolicyDTO{}, err
	}
	if rsp.JSON404 != nil {
		return approvalPolicyDTO{}, &approvalPolicyNotFoundError{orgName: orgName, appSlug: appSlug}
	}
	if rsp.JSON401 != nil {
		return approvalPolicyDTO{}, fmt.Errorf("unauthorized reading approval policy: %s", formatAPIError(rsp.JSON401))
	}
	if rsp.JSON403 != nil {
		return approvalPolicyDTO{}, fmt.Errorf("forbidden reading approval policy: %s", formatAPIError(rsp.JSON403))
	}
	if rsp.JSON500 != nil {
		return approvalPolicyDTO{}, fmt.Errorf("failed to read approval policy: %s", formatAPIError(rsp.JSON500))
	}
	if rsp.JSON200 == nil {
		return approvalPolicyDTO{}, fmt.Errorf("expected 200 response when reading approval policy for app %q in org %q, got %s", appSlug, orgName, rsp.Status())
	}

	policy, err := parseApprovalPolicyResponse(rsp.Body)
	if err != nil {
		return approvalPolicyDTO{}, err
	}
	return policy, nil
}

func upsertApprovalPolicy(ctx context.Context, client *sdk.ClientWithResponses, orgName, appSlug string, body approvalPolicyDTO) (approvalPolicyDTO, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return approvalPolicyDTO{}, err
	}
	rsp, err := client.UpsertApprovalPolicyWithBodyWithResponse(ctx, orgName, appSlug, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return approvalPolicyDTO{}, err
	}
	if rsp.JSON404 != nil {
		return approvalPolicyDTO{}, &approvalPolicyNotFoundError{orgName: orgName, appSlug: appSlug}
	}
	if rsp.JSON400 != nil {
		return approvalPolicyDTO{}, fmt.Errorf("failed to upsert approval policy: %s", formatAPIError(rsp.JSON400))
	}
	if rsp.JSON401 != nil {
		return approvalPolicyDTO{}, fmt.Errorf("unauthorized upserting approval policy: %s", formatAPIError(rsp.JSON401))
	}
	if rsp.JSON403 != nil {
		return approvalPolicyDTO{}, fmt.Errorf("forbidden upserting approval policy: %s", formatAPIError(rsp.JSON403))
	}
	if rsp.JSON500 != nil {
		return approvalPolicyDTO{}, fmt.Errorf("failed to upsert approval policy: %s", formatAPIError(rsp.JSON500))
	}
	if rsp.JSON200 == nil {
		return approvalPolicyDTO{}, fmt.Errorf("expected 200 response when upserting approval policy for app %q in org %q, got %s", appSlug, orgName, rsp.Status())
	}

	policy, err := parseApprovalPolicyResponse(rsp.Body)
	if err != nil {
		return approvalPolicyDTO{}, err
	}
	return policy, nil
}

func parseApprovalPolicyResponse(body []byte) (approvalPolicyDTO, error) {
	var response approvalPolicyAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return approvalPolicyDTO{}, err
	}
	return response.Data, nil
}

func buildUpsertApprovalPolicyBody(ctx context.Context, client *sdk.ClientWithResponses, plan JobApprovalPolicyResourceModel) (approvalPolicyDTO, map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	envNamesByID := map[string]string{}
	envIDsByName := map[string]string{}
	if len(plan.EnvironmentRules) > 0 {
		var err error
		envNamesByID, envIDsByName, err = approvalPolicyEnvironmentNameMaps(ctx, client, plan)
		if err != nil {
			diags.AddError("Failed To Resolve Approval Policy Environments", err.Error())
			return approvalPolicyDTO{}, nil, diags
		}
	}

	return buildUpsertApprovalPolicyBodyWithEnvironmentNames(ctx, plan, envNamesByID, envIDsByName)
}

func buildUpsertApprovalPolicyBodyWithEnvironmentNames(ctx context.Context, plan JobApprovalPolicyResourceModel, envNamesByID, envIDsByName map[string]string) (approvalPolicyDTO, map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	defaultRule, ruleDiags := buildUpsertApprovalRule(ctx, plan.DefaultRule)
	diags.Append(ruleDiags...)
	if diags.HasError() {
		return approvalPolicyDTO{}, nil, diags
	}

	envRules := make([]approvalPolicyEnvironmentDTO, 0, len(plan.EnvironmentRules))
	for _, rule := range plan.EnvironmentRules {
		envID, err := parseApplicationID(rule.EnvironmentID.ValueString())
		if err != nil {
			diags.AddError("Invalid Environment ID", err.Error())
			return approvalPolicyDTO{}, nil, diags
		}
		envName, ok := envNamesByID[envID.String()]
		if !ok {
			diags.AddError("Invalid Environment ID", fmt.Sprintf("environment %q was not found in app %q", envID.String(), plan.AppSlug.ValueString()))
			return approvalPolicyDTO{}, nil, diags
		}
		upsertRule, ruleDiags := buildUpsertApprovalRule(ctx, JobApprovalPolicyRuleModel{
			RequiredApprovals: rule.RequiredApprovals,
			Users:             rule.Users,
			Groups:            rule.Groups,
		})
		diags.Append(ruleDiags...)
		if diags.HasError() {
			return approvalPolicyDTO{}, nil, diags
		}
		envRules = append(envRules, approvalPolicyEnvironmentDTO{
			Environment:       envName,
			RequiredApprovals: upsertRule.RequiredApprovals,
			Approvers:         upsertRule.Approvers,
		})
	}

	return approvalPolicyDTO{
		DefaultRule:      defaultRule,
		EnvironmentRules: envRules,
	}, envIDsByName, diags
}

func buildUpsertApprovalRule(ctx context.Context, rule JobApprovalPolicyRuleModel) (approvalPolicyRuleDTO, diag.Diagnostics) {
	var diags diag.Diagnostics

	users, userDiags := terraformListToStringSlice(ctx, rule.Users)
	diags.Append(userDiags...)
	groups, groupDiags := terraformListToStringSlice(ctx, rule.Groups)
	diags.Append(groupDiags...)
	if diags.HasError() {
		return approvalPolicyRuleDTO{}, diags
	}
	if users == nil {
		users = []string{}
	}
	if groups == nil {
		groups = []string{}
	}

	return approvalPolicyRuleDTO{
		RequiredApprovals: int(rule.RequiredApprovals.ValueInt64()),
		Approvers: approvalPolicyApproversDTO{
			Users:  users,
			Groups: groups,
		},
	}, diags
}

func stateFromApprovalPolicy(ctx context.Context, base JobApprovalPolicyResourceModel, policy approvalPolicyDTO, envIDsByName map[string]string) (JobApprovalPolicyResourceModel, diag.Diagnostics) {
	defaultRule, diags := stateFromApprovalRule(ctx, base.DefaultRule, policy.DefaultRule)
	if diags.HasError() {
		return JobApprovalPolicyResourceModel{}, diags
	}

	envRules := make([]JobApprovalPolicyEnvironmentRuleModel, 0, len(policy.EnvironmentRules))
	for _, rule := range policy.EnvironmentRules {
		envID := envIDsByName[strings.ToLower(strings.TrimSpace(rule.Environment))]
		if envID == "" {
			envID = approvalPolicyEnvironmentIDFromBase(base, rule.Environment)
		}
		baseRule := approvalPolicyEnvironmentRuleFromBase(base, envID)
		ruleState, ruleDiags := stateFromApprovalRule(ctx, baseRule, approvalPolicyRuleDTO{
			RequiredApprovals: rule.RequiredApprovals,
			Approvers:         rule.Approvers,
		})
		diags.Append(ruleDiags...)
		if diags.HasError() {
			return JobApprovalPolicyResourceModel{}, diags
		}
		envRules = append(envRules, JobApprovalPolicyEnvironmentRuleModel{
			EnvironmentID:     types.StringValue(envID),
			RequiredApprovals: ruleState.RequiredApprovals,
			Users:             ruleState.Users,
			Groups:            ruleState.Groups,
		})
	}

	return JobApprovalPolicyResourceModel{
		ID:               types.StringValue(approvalPolicyID(base.OrgName.ValueString(), base.AppSlug.ValueString())),
		OrgName:          base.OrgName,
		AppSlug:          base.AppSlug,
		Version:          types.Int64Value(int64(policy.Version)),
		DefaultRule:      defaultRule,
		EnvironmentRules: envRules,
	}, diags
}

func stateFromApprovalRule(ctx context.Context, base JobApprovalPolicyRuleModel, rule approvalPolicyRuleDTO) (JobApprovalPolicyRuleModel, diag.Diagnostics) {
	users, diags := types.ListValueFrom(ctx, types.StringType, rule.Approvers.Users)
	if diags.HasError() {
		return JobApprovalPolicyRuleModel{}, diags
	}
	if !base.Users.IsNull() && !base.Users.IsUnknown() {
		users = base.Users
	}

	groupValues, groupDiags := types.ListValueFrom(ctx, types.StringType, rule.Approvers.Groups)
	diags.Append(groupDiags...)
	if diags.HasError() {
		return JobApprovalPolicyRuleModel{}, diags
	}
	if !base.Groups.IsNull() && !base.Groups.IsUnknown() {
		groupValues = base.Groups
	}

	return JobApprovalPolicyRuleModel{
		RequiredApprovals: types.Int64Value(int64(rule.RequiredApprovals)),
		Users:             users,
		Groups:            groupValues,
	}, diags
}

func approvalPolicyEnvironmentIDsByName(ctx context.Context, client *sdk.ClientWithResponses, plan JobApprovalPolicyResourceModel) (map[string]string, error) {
	_, idsByName, err := approvalPolicyEnvironmentNameMaps(ctx, client, plan)
	return idsByName, err
}

func approvalPolicyEnvironmentNameMaps(ctx context.Context, client *sdk.ClientWithResponses, plan JobApprovalPolicyResourceModel) (map[string]string, map[string]string, error) {
	namesByID := map[string]string{}
	idsByName := map[string]string{}

	rsp, err := client.ListEnvironmentsWithResponse(ctx, plan.OrgName.ValueString(), plan.AppSlug.ValueString())
	if err != nil {
		return nil, nil, err
	}
	if rsp.JSON404 != nil {
		return nil, nil, fmt.Errorf("app %q in org %q not found", plan.AppSlug.ValueString(), plan.OrgName.ValueString())
	}
	if rsp.JSON200 == nil {
		return nil, nil, fmt.Errorf("expected 200 response when listing environments for approval policy, got %s", rsp.Status())
	}

	for _, env := range rsp.JSON200.Data.Items {
		namesByID[env.Id.String()] = env.Name
		idsByName[strings.ToLower(strings.TrimSpace(env.Name))] = env.Id.String()
	}
	return namesByID, idsByName, nil
}

func approvalPolicyEnvironmentIDFromBase(base JobApprovalPolicyResourceModel, environmentName string) string {
	if len(base.EnvironmentRules) == 1 {
		return base.EnvironmentRules[0].EnvironmentID.ValueString()
	}
	return ""
}

func approvalPolicyEnvironmentRuleFromBase(base JobApprovalPolicyResourceModel, environmentID string) JobApprovalPolicyRuleModel {
	for _, rule := range base.EnvironmentRules {
		if rule.EnvironmentID.ValueString() == environmentID {
			return JobApprovalPolicyRuleModel{
				RequiredApprovals: rule.RequiredApprovals,
				Users:             rule.Users,
				Groups:            rule.Groups,
			}
		}
	}
	if len(base.EnvironmentRules) == 1 {
		rule := base.EnvironmentRules[0]
		return JobApprovalPolicyRuleModel{
			RequiredApprovals: rule.RequiredApprovals,
			Users:             rule.Users,
			Groups:            rule.Groups,
		}
	}
	return JobApprovalPolicyRuleModel{}
}

func approvalPolicyID(orgName, appSlug string) string {
	return strings.Join([]string{orgName, appSlug}, "/")
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
