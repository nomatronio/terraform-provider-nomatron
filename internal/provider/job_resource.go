package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &JobResource{}
var _ resource.ResourceWithImportState = &JobResource{}

type JobResource struct {
	client *sdk.ClientWithResponses
}

func NewJobResource() resource.Resource {
	return &JobResource{}
}

type JobResourceModel struct {
	OrgName          types.String `tfsdk:"org_name"`
	AppSlug          types.String `tfsdk:"app_slug"`
	ID               types.String `tfsdk:"id"`
	Slug             types.String `tfsdk:"slug"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	ClusterID        types.String `tfsdk:"cluster_id"`
	DefaultNamespace types.String `tfsdk:"default_namespace"`
	JobspecPath      types.String `tfsdk:"jobspec_path"`
	JobspecType      types.String `tfsdk:"jobspec_type"`
	IsPrimary        types.Bool   `tfsdk:"is_primary"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *JobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (r *JobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron application job resource.",
		Attributes: map[string]schema.Attribute{
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
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Job ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Stable job slug. If omitted, the server generates one.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Job display name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Job description.",
			},
			"cluster_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional cluster override for this job.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_namespace": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Default Nomad namespace for this job.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"jobspec_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to the job spec within the repository.",
			},
			"jobspec_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Job spec source type.",
			},
			"is_primary": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this is the primary job for the application.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the job was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the job was last updated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *JobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *JobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan JobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the job resource.",
		)
		return
	}

	body, diags := buildCreateAppJobBody(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.CreateAppJobWithResponse(ctx, plan.OrgName.ValueString(), plan.AppSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Job", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Job", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Application Or Cluster Not Found", formatAPIError(rsp.JSON404))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Create Job", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating job, got %s.", rsp.Status()),
		)
		return
	}

	state := stateFromAppJob(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *JobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state JobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the job resource.",
		)
		return
	}

	job, err := readAppJob(ctx, r.client, state.OrgName.ValueString(), state.AppSlug.ValueString(), state.Slug.ValueString())
	if err != nil {
		if isAppJobNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Job", err.Error())
		return
	}

	newState := stateFromAppJob(state, job)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *JobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan JobResourceModel
	var state JobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the job resource.",
		)
		return
	}

	body, diags := buildUpdateAppJobBody(plan, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.UpdateAppJobWithResponse(ctx, state.OrgName.ValueString(), state.AppSlug.ValueString(), state.Slug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Job", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Job", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Update Job", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating job %q for app %q in org %q, got %s.", state.Slug.ValueString(), state.AppSlug.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	newState := stateFromAppJob(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *JobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state JobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the job resource.",
		)
		return
	}

	rsp, err := r.client.DeleteAppJobWithResponse(ctx, state.OrgName.ValueString(), state.AppSlug.ValueString(), state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Job", err.Error())
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
		resp.Diagnostics.AddError("Failed To Delete Job", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting job %q for app %q in org %q, got %s.", state.Slug.ValueString(), state.AppSlug.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *JobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgName, appSlug, jobSlug, err := parseJobImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_slug"), appSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), jobSlug)...)
}

type appJobNotFoundError struct {
	orgName string
	appSlug string
	jobSlug string
}

func (e *appJobNotFoundError) Error() string {
	return fmt.Sprintf("job %q for app %q in org %q not found", e.jobSlug, e.appSlug, e.orgName)
}

func isAppJobNotFound(err error) bool {
	_, ok := err.(*appJobNotFoundError)
	return ok
}

func parseJobImportID(raw string) (orgName, appSlug, jobSlug string, err error) {
	parts := strings.SplitN(raw, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("expected import identifier in the format `org_name/app_slug/job_slug`")
	}
	return parts[0], parts[1], parts[2], nil
}

func readAppJob(ctx context.Context, client *sdk.ClientWithResponses, orgName, appSlug, jobSlug string) (sdk.AppJob, error) {
	rsp, err := client.GetAppJobWithResponse(ctx, orgName, appSlug, jobSlug)
	if err != nil {
		return sdk.AppJob{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.AppJob{}, &appJobNotFoundError{orgName: orgName, appSlug: appSlug, jobSlug: jobSlug}
	}
	if rsp.JSON200 == nil {
		return sdk.AppJob{}, fmt.Errorf("expected 200 response when reading job %q for app %q in org %q, got %s", jobSlug, appSlug, orgName, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func buildCreateAppJobBody(plan JobResourceModel) (sdk.CreateAppJobJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := sdk.CreateAppJobJSONRequestBody{
		JobspecPath: plan.JobspecPath.ValueString(),
		JobspecType: sdk.CreateAppJobRequestJobspecType(plan.JobspecType.ValueString()),
		Name:        plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}
	if !plan.ClusterID.IsNull() && !plan.ClusterID.IsUnknown() && plan.ClusterID.ValueString() != "" {
		clusterID, err := parseApplicationID(plan.ClusterID.ValueString())
		if err != nil {
			diags.AddError("Invalid Cluster ID", err.Error())
			return body, diags
		}
		body.ClusterId = &clusterID
	}
	if !plan.DefaultNamespace.IsNull() && !plan.DefaultNamespace.IsUnknown() && plan.DefaultNamespace.ValueString() != "" {
		defaultNamespace := plan.DefaultNamespace.ValueString()
		body.DefaultNamespace = &defaultNamespace
	}
	if !plan.IsPrimary.IsNull() && !plan.IsPrimary.IsUnknown() {
		isPrimary := plan.IsPrimary.ValueBool()
		body.IsPrimary = &isPrimary
	}
	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() && plan.Slug.ValueString() != "" {
		slug := plan.Slug.ValueString()
		body.Slug = &slug
	}

	return body, diags
}

func buildUpdateAppJobBody(plan, state JobResourceModel) (sdk.UpdateAppJobJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := sdk.UpdateAppJobJSONRequestBody{}

	if stringValueChanged(plan.Name, state.Name) {
		name := plan.Name.ValueString()
		body.Name = &name
	}
	if stringValueChanged(plan.Description, state.Description) && !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}
	if stringValueChanged(plan.ClusterID, state.ClusterID) && !plan.ClusterID.IsNull() && !plan.ClusterID.IsUnknown() {
		clusterID, err := parseApplicationID(plan.ClusterID.ValueString())
		if err != nil {
			diags.AddError("Invalid Cluster ID", err.Error())
			return body, diags
		}
		body.ClusterId = &clusterID
	}
	if stringValueChanged(plan.DefaultNamespace, state.DefaultNamespace) && !plan.DefaultNamespace.IsNull() && !plan.DefaultNamespace.IsUnknown() {
		defaultNamespace := plan.DefaultNamespace.ValueString()
		body.DefaultNamespace = &defaultNamespace
	}
	if stringValueChanged(plan.JobspecPath, state.JobspecPath) {
		jobspecPath := plan.JobspecPath.ValueString()
		body.JobspecPath = &jobspecPath
	}
	if stringValueChanged(plan.JobspecType, state.JobspecType) {
		jobspecType := sdk.UpdateAppJobRequestJobspecType(plan.JobspecType.ValueString())
		body.JobspecType = &jobspecType
	}
	if boolValueChanged(plan.IsPrimary, state.IsPrimary) && !plan.IsPrimary.IsNull() && !plan.IsPrimary.IsUnknown() {
		isPrimary := plan.IsPrimary.ValueBool()
		body.IsPrimary = &isPrimary
	}

	return body, diags
}

func stateFromAppJob(base JobResourceModel, job sdk.AppJob) JobResourceModel {
	description := base.Description
	if description.IsUnknown() {
		description = types.StringNull()
	}
	if job.Description != nil {
		description = types.StringValue(*job.Description)
	}

	clusterID := base.ClusterID
	if clusterID.IsUnknown() {
		clusterID = types.StringNull()
	}
	if job.ClusterID != nil {
		clusterID = types.StringValue(job.ClusterID.String())
	}

	defaultNamespace := base.DefaultNamespace
	if defaultNamespace.IsUnknown() {
		defaultNamespace = types.StringNull()
	}
	if job.DefaultNamespace != "" {
		defaultNamespace = types.StringValue(job.DefaultNamespace)
	}

	createdAt := types.StringNull()
	if !job.CreatedAt.IsZero() {
		createdAt = types.StringValue(job.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !job.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(job.UpdatedAt.Format(time.RFC3339))
	}

	return JobResourceModel{
		OrgName:          base.OrgName,
		AppSlug:          base.AppSlug,
		ID:               types.StringValue(job.ID.String()),
		Slug:             types.StringValue(job.Slug),
		Name:             types.StringValue(job.Name),
		Description:      description,
		ClusterID:        clusterID,
		DefaultNamespace: defaultNamespace,
		JobspecPath:      types.StringValue(job.JobspecPath),
		JobspecType:      types.StringValue(job.JobspecType),
		IsPrimary:        types.BoolValue(job.IsPrimary),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}
