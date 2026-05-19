package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ resource.Resource = &ApplicationResource{}
var _ resource.ResourceWithImportState = &ApplicationResource{}

type ApplicationResource struct {
	client *sdk.ClientWithResponses
}

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

type ApplicationResourceModel struct {
	OrgName        types.String `tfsdk:"org_name"`
	ID             types.String `tfsdk:"id"`
	Slug           types.String `tfsdk:"slug"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	ClusterID      types.String `tfsdk:"cluster_id"`
	RepoURL        types.String `tfsdk:"repo_url"`
	GitProvider    types.String `tfsdk:"git_provider"`
	Ref            types.String `tfsdk:"ref"`
	TriggerMode    types.String `tfsdk:"trigger_mode"`
	TagPatternType types.String `tfsdk:"tag_pattern_type"`
	TagPattern     types.String `tfsdk:"tag_pattern"`
	AutoPlan       types.Bool   `tfsdk:"auto_plan"`
	AutoApply      types.Bool   `tfsdk:"auto_apply"`
	VcsGitHubID    types.String `tfsdk:"vcs_github_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	UpdatedBy      types.String `tfsdk:"updated_by"`
}

func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron application resource.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Stable application slug. If omitted, the server generates one.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application display name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Application description.",
			},
			"cluster_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Nomad cluster ID backing the application.",
			},
			"repo_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source repository URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"git_provider": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Git provider for the repository, such as `github`.",
				PlanModifiers: []planmodifier.String{
					lowerCaseString(),
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ref": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Branch ref tracked when `trigger_mode` is `branch_commit`. Defaults to `main` when omitted.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"trigger_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Deployment trigger mode. Use `branch_commit` to deploy from branch pushes, or `tag` to deploy from matching Git tags. Defaults to `branch_commit`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag_pattern_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Tag matching style for tag-based triggers: `semver`, `prefix`, `suffix`, or `custom_regex`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag_pattern": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Prefix, suffix, or regular expression used when `trigger_mode` is `tag`. Leave unset for `semver`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_plan": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether Nomatron automatically plans changes for this application.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_apply": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether Nomatron automatically applies approved changes for this application.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"vcs_github_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional GitHub App integration ID for repository sync.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the application was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the application was last updated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Actor ID that last updated the application, when available.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the application resource.",
		)
		return
	}

	clusterID, err := parseAgentID(plan.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Cluster ID", err.Error())
		return
	}

	body := sdk.CreateAppJSONRequestBody{
		ClusterId: clusterID,
		Name:      plan.Name.ValueString(),
		RepoUrl:   plan.RepoURL.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}
	if !plan.GitProvider.IsNull() && !plan.GitProvider.IsUnknown() && plan.GitProvider.ValueString() != "" {
		gitProvider := sdk.GitProvider(normalizeProviderString(plan.GitProvider.ValueString()))
		body.GitProvider = &gitProvider
	}
	if !plan.Ref.IsNull() && !plan.Ref.IsUnknown() && plan.Ref.ValueString() != "" {
		ref := plan.Ref.ValueString()
		body.Ref = &ref
	}
	if diags := applyApplicationTriggerCreate(plan, &body); diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}
	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() && plan.Slug.ValueString() != "" {
		slug := plan.Slug.ValueString()
		body.Slug = &slug
	}
	if !plan.AutoPlan.IsNull() && !plan.AutoPlan.IsUnknown() {
		autoPlan := plan.AutoPlan.ValueBool()
		body.AutoPlan = &autoPlan
	}
	if !plan.AutoApply.IsNull() && !plan.AutoApply.IsUnknown() {
		autoApply := plan.AutoApply.ValueBool()
		body.AutoApply = &autoApply
	}
	if !plan.VcsGitHubID.IsNull() && !plan.VcsGitHubID.IsUnknown() && plan.VcsGitHubID.ValueString() != "" {
		vcsGitHubID, parseErr := parseAgentID(plan.VcsGitHubID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid GitHub Integration ID", parseErr.Error())
			return
		}
		body.VcsGithubId = &vcsGitHubID
	}

	rsp, err := r.client.CreateAppWithResponse(ctx, plan.OrgName.ValueString(), nil, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Application", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Application", formatAPIError(rsp.JSON400))
		return
	}
	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatAPIError(rsp.JSON401))
		return
	}
	if rsp.JSON403 != nil {
		resp.Diagnostics.AddError("Forbidden", formatAPIError(rsp.JSON403))
		return
	}
	if rsp.JSON404 != nil {
		resp.Diagnostics.AddError("Organization Or Cluster Not Found", formatAPIError(rsp.JSON404))
		return
	}
	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Application Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Create Application", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating application, got %s.", rsp.Status()),
		)
		return
	}

	state := stateFromApplication(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the application resource.",
		)
		return
	}

	app, err := readApplication(ctx, r.client, state.OrgName.ValueString(), state.Slug.ValueString())
	if err != nil {
		if isApplicationNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Application", err.Error())
		return
	}

	newState := stateFromApplication(state, app)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationResourceModel
	var state ApplicationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the application resource.",
		)
		return
	}

	body := sdk.UpdateAppJSONRequestBody{}

	if stringValueChanged(plan.Name, state.Name) {
		name := plan.Name.ValueString()
		body.Name = &name
	}
	if stringValueChanged(plan.Description, state.Description) && !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}
	if stringValueChanged(plan.Ref, state.Ref) && !plan.Ref.IsNull() && !plan.Ref.IsUnknown() {
		ref := plan.Ref.ValueString()
		body.Ref = &ref
	}
	if diags := applyApplicationTriggerUpdate(plan, state, &body); diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}
	if boolValueChanged(plan.AutoPlan, state.AutoPlan) && !plan.AutoPlan.IsNull() && !plan.AutoPlan.IsUnknown() {
		autoPlan := plan.AutoPlan.ValueBool()
		body.AutoPlan = &autoPlan
	}
	if boolValueChanged(plan.AutoApply, state.AutoApply) && !plan.AutoApply.IsNull() && !plan.AutoApply.IsUnknown() {
		autoApply := plan.AutoApply.ValueBool()
		body.AutoApply = &autoApply
	}
	if stringValueChanged(plan.ClusterID, state.ClusterID) && !plan.ClusterID.IsNull() && !plan.ClusterID.IsUnknown() {
		clusterID, err := parseAgentID(plan.ClusterID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Cluster ID", err.Error())
			return
		}
		body.ClusterId = &clusterID
	}

	rsp, err := r.client.UpdateAppWithResponse(ctx, state.OrgName.ValueString(), state.Slug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Application", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Application", formatAPIError(rsp.JSON400))
		return
	}
	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatAPIError(rsp.JSON401))
		return
	}
	if rsp.JSON403 != nil {
		resp.Diagnostics.AddError("Forbidden", formatAPIError(rsp.JSON403))
		return
	}
	if rsp.JSON404 != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Update Application", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating application %q in org %q, got %s.", state.Slug.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	newState := stateFromApplication(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the application resource.",
		)
		return
	}

	rsp, err := r.client.DeleteAppWithResponse(ctx, state.OrgName.ValueString(), state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Application", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		return
	}
	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatAPIError(rsp.JSON401))
		return
	}
	if rsp.JSON403 != nil {
		resp.Diagnostics.AddError("Forbidden", formatAPIError(rsp.JSON403))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Delete Application", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting application %q in org %q, got %s.", state.Slug.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *ApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgName, slug, err := parseApplicationImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), slug)...)
}

type applicationNotFoundError struct {
	orgName string
	slug    string
}

func (e *applicationNotFoundError) Error() string {
	return fmt.Sprintf("application %q in org %q not found", e.slug, e.orgName)
}

func isApplicationNotFound(err error) bool {
	_, ok := err.(*applicationNotFoundError)
	return ok
}

func parseApplicationImportID(raw string) (orgName, slug string, err error) {
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected import identifier in the format `org_name/app_slug`")
	}
	return parts[0], parts[1], nil
}

func readApplication(ctx context.Context, client *sdk.ClientWithResponses, orgName, slug string) (sdk.App, error) {
	rsp, err := client.GetAppWithResponse(ctx, orgName, slug)
	if err != nil {
		return sdk.App{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.App{}, &applicationNotFoundError{orgName: orgName, slug: slug}
	}
	if rsp.JSON200 == nil {
		return sdk.App{}, fmt.Errorf("expected 200 response when reading application %q in org %q, got %s", slug, orgName, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func stateFromApplication(base ApplicationResourceModel, app sdk.App) ApplicationResourceModel {
	description := base.Description
	if description.IsUnknown() {
		description = types.StringNull()
	}
	if app.Description != nil {
		description = types.StringValue(*app.Description)
	}

	ref := base.Ref
	if ref.IsUnknown() {
		ref = types.StringNull()
	}
	if app.Ref != nil {
		ref = types.StringValue(*app.Ref)
	}
	triggerMode := base.TriggerMode
	if triggerMode.IsUnknown() {
		triggerMode = types.StringNull()
	}
	if app.TriggerMode != nil {
		triggerMode = types.StringValue(string(*app.TriggerMode))
	}
	tagPatternType := base.TagPatternType
	if tagPatternType.IsUnknown() {
		tagPatternType = types.StringNull()
	}
	if app.TagPatternType != nil {
		tagPatternType = types.StringValue(string(*app.TagPatternType))
	}
	tagPattern := base.TagPattern
	if tagPattern.IsUnknown() {
		tagPattern = types.StringNull()
	}
	if app.TagPattern != nil {
		tagPattern = types.StringValue(*app.TagPattern)
	}

	vcsGitHubID := base.VcsGitHubID
	if vcsGitHubID.IsUnknown() {
		vcsGitHubID = types.StringNull()
	}
	if app.VcsGithubId != nil {
		vcsGitHubID = types.StringValue(app.VcsGithubId.String())
	}

	createdAt := types.StringNull()
	if !app.CreatedAt.IsZero() {
		createdAt = types.StringValue(app.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !app.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(app.UpdatedAt.Format(time.RFC3339))
	}

	updatedBy := types.StringNull()
	if app.UpdatedBy != nil {
		updatedBy = types.StringValue(app.UpdatedBy.String())
	}

	clusterID := base.ClusterID
	if clusterID.IsUnknown() {
		clusterID = types.StringNull()
	}
	if app.ClusterId != uuid.Nil {
		clusterID = types.StringValue(app.ClusterId.String())
	}

	repoURL := base.RepoURL
	if repoURL.IsUnknown() {
		repoURL = types.StringNull()
	}
	if app.RepoUrl != "" {
		repoURL = types.StringValue(app.RepoUrl)
	}

	gitProvider := base.GitProvider
	if gitProvider.IsUnknown() {
		gitProvider = types.StringNull()
	} else if !gitProvider.IsNull() {
		gitProvider = types.StringValue(normalizeProviderString(gitProvider.ValueString()))
	}
	if app.GitProvider != "" {
		gitProvider = types.StringValue(normalizeProviderString(string(app.GitProvider)))
	}

	return ApplicationResourceModel{
		OrgName:        base.OrgName,
		ID:             types.StringValue(app.Id.String()),
		Slug:           types.StringValue(app.Slug),
		Name:           types.StringValue(app.Name),
		Description:    description,
		ClusterID:      clusterID,
		RepoURL:        repoURL,
		GitProvider:    gitProvider,
		Ref:            ref,
		TriggerMode:    triggerMode,
		TagPatternType: tagPatternType,
		TagPattern:     tagPattern,
		AutoPlan:       types.BoolValue(app.AutoPlan),
		AutoApply:      types.BoolValue(app.AutoApply),
		VcsGitHubID:    vcsGitHubID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		UpdatedBy:      updatedBy,
	}
}

func applyApplicationTriggerCreate(plan ApplicationResourceModel, body *sdk.CreateAppJSONRequestBody) diag.Diagnostics {
	var diags diag.Diagnostics
	mode := strings.TrimSpace(plan.TriggerMode.ValueString())
	if !plan.TriggerMode.IsNull() && !plan.TriggerMode.IsUnknown() && mode != "" {
		triggerMode := sdk.DeploymentTriggerMode(mode)
		body.TriggerMode = &triggerMode
	}
	if !plan.TagPatternType.IsNull() && !plan.TagPatternType.IsUnknown() && strings.TrimSpace(plan.TagPatternType.ValueString()) != "" {
		patternType := sdk.TagPatternType(strings.TrimSpace(plan.TagPatternType.ValueString()))
		body.TagPatternType = &patternType
	}
	if !plan.TagPattern.IsNull() && !plan.TagPattern.IsUnknown() && strings.TrimSpace(plan.TagPattern.ValueString()) != "" {
		pattern := strings.TrimSpace(plan.TagPattern.ValueString())
		body.TagPattern = &pattern
	}
	if mode == string(sdk.Tag) && !plan.Ref.IsNull() && strings.TrimSpace(plan.Ref.ValueString()) != "" && strings.TrimSpace(plan.Ref.ValueString()) != "main" {
		diags.AddError("Invalid Application Trigger Configuration", "`ref` is only used for branch_commit triggers. Remove `ref` or set `trigger_mode` to `branch_commit`.")
	}
	return diags
}

func applyApplicationTriggerUpdate(plan, state ApplicationResourceModel, body *sdk.UpdateAppJSONRequestBody) diag.Diagnostics {
	var diags diag.Diagnostics
	if stringValueChanged(plan.TriggerMode, state.TriggerMode) && !plan.TriggerMode.IsNull() && !plan.TriggerMode.IsUnknown() {
		triggerMode := sdk.DeploymentTriggerMode(strings.TrimSpace(plan.TriggerMode.ValueString()))
		body.TriggerMode = &triggerMode
	}
	if stringValueChanged(plan.TagPatternType, state.TagPatternType) && !plan.TagPatternType.IsNull() && !plan.TagPatternType.IsUnknown() {
		patternType := sdk.TagPatternType(strings.TrimSpace(plan.TagPatternType.ValueString()))
		body.TagPatternType = &patternType
	}
	if stringValueChanged(plan.TagPattern, state.TagPattern) && !plan.TagPattern.IsNull() && !plan.TagPattern.IsUnknown() {
		pattern := strings.TrimSpace(plan.TagPattern.ValueString())
		body.TagPattern = &pattern
	}
	mode := strings.TrimSpace(plan.TriggerMode.ValueString())
	if mode == string(sdk.Tag) && !plan.Ref.IsNull() && strings.TrimSpace(plan.Ref.ValueString()) != "" && strings.TrimSpace(plan.Ref.ValueString()) != "main" {
		diags.AddError("Invalid Application Trigger Configuration", "`ref` is only used for branch_commit triggers. Remove `ref` or set `trigger_mode` to `branch_commit`.")
	}
	return diags
}

func parseApplicationID(raw string) (openapi_types.UUID, error) {
	return parseAgentID(raw)
}
