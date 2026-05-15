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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &EnvironmentResource{}
var _ resource.ResourceWithImportState = &EnvironmentResource{}

type EnvironmentResource struct {
	client *sdk.ClientWithResponses
}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

type EnvironmentResourceModel struct {
	OrgName   types.String `tfsdk:"org_name"`
	AppSlug   types.String `tfsdk:"app_slug"`
	JobSlug   types.String `tfsdk:"job_slug"`
	ID        types.String `tfsdk:"id"`
	Slug      types.String `tfsdk:"slug"`
	Name      types.String `tfsdk:"name"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Namespace types.String `tfsdk:"namespace"`
	Priority  types.Int64  `tfsdk:"priority"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *EnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron application environment resource.",
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
				MarkdownDescription: "Application slug that owns the environment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"job_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Job slug that owns the environment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Environment ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Stable environment slug. If omitted, the server generates one.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Environment display name.",
			},
			"cluster_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Nomad cluster ID used by the environment.",
			},
			"namespace": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Nomad namespace for the environment.",
			},
			"priority": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Relative environment priority used by Nomatron.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the environment was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the environment was last updated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *EnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the environment resource.",
		)
		return
	}

	body, diags := buildCreateEnvironmentBody(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.CreateEnvironmentWithResponse(ctx, plan.OrgName.ValueString(), plan.AppSlug.ValueString(), plan.JobSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Environment", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Environment", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Job Or Cluster Not Found", formatAPIError(rsp.JSON404))
		return
	}
	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Environment Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Create Environment", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating environment, got %s.", rsp.Status()),
		)
		return
	}

	environment := rsp.JSON201.Data
	if priorityUpdateRequired(plan, environment) {
		updated, err := updateEnvironmentPriority(ctx, r.client, plan, environment)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Set Environment Priority", err.Error())
			return
		}
		environment = updated
	}

	state := stateFromEnvironment(plan, environment)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the environment resource.",
		)
		return
	}

	env, err := readEnvironment(ctx, r.client, state.OrgName.ValueString(), state.AppSlug.ValueString(), state.JobSlug.ValueString(), state.Slug.ValueString())
	if err != nil {
		if isEnvironmentNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Environment", err.Error())
		return
	}

	newState := stateFromEnvironment(state, env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EnvironmentResourceModel
	var state EnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the environment resource.",
		)
		return
	}

	body, diags := buildUpdateEnvironmentBody(plan, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.UpdateEnvironmentWithResponse(ctx, state.OrgName.ValueString(), state.AppSlug.ValueString(), state.JobSlug.ValueString(), state.Slug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Environment", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Environment", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Update Environment", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating environment %q for job %q in app %q in org %q, got %s.", state.Slug.ValueString(), state.JobSlug.ValueString(), state.AppSlug.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	newState := stateFromEnvironment(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the environment resource.",
		)
		return
	}

	rsp, err := r.client.DeleteEnvironmentWithResponse(ctx, state.OrgName.ValueString(), state.AppSlug.ValueString(), state.JobSlug.ValueString(), state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Environment", err.Error())
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
		resp.Diagnostics.AddError("Failed To Delete Environment", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting environment %q for job %q in app %q in org %q, got %s.", state.Slug.ValueString(), state.JobSlug.ValueString(), state.AppSlug.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgName, appSlug, jobSlug, environmentSlug, err := parseEnvironmentImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_slug"), appSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("job_slug"), jobSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), environmentSlug)...)
}

type environmentNotFoundError struct {
	orgName         string
	appSlug         string
	jobSlug         string
	environmentSlug string
}

func (e *environmentNotFoundError) Error() string {
	return fmt.Sprintf("environment %q for job %q in app %q in org %q not found", e.environmentSlug, e.jobSlug, e.appSlug, e.orgName)
}

func isEnvironmentNotFound(err error) bool {
	_, ok := err.(*environmentNotFoundError)
	return ok
}

func parseEnvironmentImportID(raw string) (orgName, appSlug, jobSlug, environmentSlug string, err error) {
	parts := strings.SplitN(raw, "/", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return "", "", "", "", fmt.Errorf("expected import identifier in the format `org_name/app_slug/job_slug/environment_slug`")
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}

func readEnvironment(ctx context.Context, client *sdk.ClientWithResponses, orgName, appSlug, jobSlug, environmentSlug string) (sdk.Environment, error) {
	rsp, err := client.GetEnvironmentWithResponse(ctx, orgName, appSlug, jobSlug, environmentSlug)
	if err != nil {
		return sdk.Environment{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.Environment{}, &environmentNotFoundError{
			orgName:         orgName,
			appSlug:         appSlug,
			jobSlug:         jobSlug,
			environmentSlug: environmentSlug,
		}
	}
	if rsp.JSON200 == nil {
		return sdk.Environment{}, fmt.Errorf("expected 200 response when reading environment %q for job %q in app %q in org %q, got %s", environmentSlug, jobSlug, appSlug, orgName, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func buildCreateEnvironmentBody(plan EnvironmentResourceModel) (sdk.CreateEnvironmentJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	clusterID, err := parseApplicationID(plan.ClusterID.ValueString())
	if err != nil {
		diags.AddError("Invalid Cluster ID", err.Error())
		return sdk.CreateEnvironmentJSONRequestBody{}, diags
	}

	body := sdk.CreateEnvironmentJSONRequestBody{
		ClusterId: clusterID,
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
		Priority:  int(plan.Priority.ValueInt64()),
		Slug:      plan.Slug.ValueString(),
	}

	return body, diags
}

func buildUpdateEnvironmentBody(plan, state EnvironmentResourceModel) (sdk.UpdateEnvironmentJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := sdk.UpdateEnvironmentJSONRequestBody{}

	if stringValueChanged(plan.Name, state.Name) {
		name := plan.Name.ValueString()
		body.Name = &name
	}
	if stringValueChanged(plan.ClusterID, state.ClusterID) {
		clusterID, err := parseApplicationID(plan.ClusterID.ValueString())
		if err != nil {
			diags.AddError("Invalid Cluster ID", err.Error())
			return body, diags
		}
		body.ClusterId = &clusterID
	}
	if stringValueChanged(plan.Namespace, state.Namespace) {
		namespace := plan.Namespace.ValueString()
		body.Namespace = &namespace
	}
	if int64ValueChanged(plan.Priority, state.Priority) {
		priority := int(plan.Priority.ValueInt64())
		body.Priority = &priority
	}

	return body, diags
}

func priorityUpdateRequired(plan EnvironmentResourceModel, env sdk.Environment) bool {
	if plan.Priority.IsNull() || plan.Priority.IsUnknown() {
		return false
	}
	return env.Priority != int(plan.Priority.ValueInt64())
}

func updateEnvironmentPriority(ctx context.Context, client *sdk.ClientWithResponses, plan EnvironmentResourceModel, env sdk.Environment) (sdk.Environment, error) {
	priority := int(plan.Priority.ValueInt64())
	body := sdk.UpdateEnvironmentJSONRequestBody{
		Priority: &priority,
	}

	rsp, err := client.UpdateEnvironmentWithResponse(
		ctx,
		plan.OrgName.ValueString(),
		plan.AppSlug.ValueString(),
		plan.JobSlug.ValueString(),
		env.Slug,
		body,
	)
	if err != nil {
		return sdk.Environment{}, err
	}

	if rsp.JSON400 != nil {
		return sdk.Environment{}, fmt.Errorf("failed to update environment priority: %s", formatAPIError(rsp.JSON400))
	}
	if rsp.JSON401 != nil {
		return sdk.Environment{}, fmt.Errorf("unauthorized updating environment priority: %s", formatAPIError(rsp.JSON401))
	}
	if rsp.JSON403 != nil {
		return sdk.Environment{}, fmt.Errorf("forbidden updating environment priority: %s", formatAPIError(rsp.JSON403))
	}
	if rsp.JSON404 != nil {
		return sdk.Environment{}, fmt.Errorf("environment %q for job %q in app %q in org %q not found after create", env.Slug, plan.JobSlug.ValueString(), plan.AppSlug.ValueString(), plan.OrgName.ValueString())
	}
	if rsp.JSON500 != nil {
		return sdk.Environment{}, fmt.Errorf("failed to update environment priority: %s", formatAPIError(rsp.JSON500))
	}
	if rsp.JSON200 == nil {
		return sdk.Environment{}, fmt.Errorf("expected 200 response when updating environment priority, got %s", rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func stateFromEnvironment(base EnvironmentResourceModel, env sdk.Environment) EnvironmentResourceModel {
	createdAt := types.StringNull()
	if !env.CreatedAt.IsZero() {
		createdAt = types.StringValue(env.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !env.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(env.UpdatedAt.Format(time.RFC3339))
	}

	return EnvironmentResourceModel{
		OrgName:   base.OrgName,
		AppSlug:   base.AppSlug,
		JobSlug:   base.JobSlug,
		ID:        types.StringValue(env.Id.String()),
		Slug:      types.StringValue(env.Slug),
		Name:      types.StringValue(env.Name),
		ClusterID: types.StringValue(env.ClusterId.String()),
		Namespace: types.StringValue(env.Namespace),
		Priority:  types.Int64Value(int64(env.Priority)),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func int64ValueChanged(plan, state types.Int64) bool {
	if plan.IsNull() != state.IsNull() {
		return true
	}
	if plan.IsUnknown() || state.IsUnknown() {
		return false
	}
	if plan.IsNull() && state.IsNull() {
		return false
	}
	return plan.ValueInt64() != state.ValueInt64()
}
