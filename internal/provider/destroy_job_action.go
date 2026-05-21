package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ action.Action = &DestroyJobAction{}
var _ action.ActionWithConfigure = &DestroyJobAction{}

type DestroyJobAction struct {
	client *sdk.ClientWithResponses
}

func NewDestroyJobAction() action.Action {
	return &DestroyJobAction{}
}

type DestroyJobActionModel struct {
	OrgName types.String `tfsdk:"org_name"`
	AppSlug types.String `tfsdk:"app_slug"`
	JobSlug types.String `tfsdk:"job_slug"`
	Apply   types.Bool   `tfsdk:"apply"`
}

func (a *DestroyJobAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destroy_job"
}

func (a *DestroyJobAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Create a destroy plan for one live Nomad job managed by Nomatron. This stops and purges runtime state only; it does not delete the Nomatron job resource.",
		Attributes: map[string]actionschema.Attribute{
			"org_name": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the application.",
			},
			"app_slug": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application slug that owns the job.",
			},
			"job_slug": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Nomatron job slug whose live Nomad runtime should be stopped and purged.",
			},
			"apply": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When true, immediately queue a destroy apply if the destroy plan is ready for apply. Approval-required plans still need approval before apply can be queued.",
			},
		},
	}
}

func (a *DestroyJobAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Action Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T.", req.ProviderData),
		)
		return
	}

	a.client = client
}

func (a *DestroyJobAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data DestroyJobActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if a.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the destroy job action.",
		)
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: fmt.Sprintf("Creating destroy plan for job %q.", data.JobSlug.ValueString()),
		})
	}

	rsp, err := a.client.CreateJobDestroyPlanWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Destroy Plan", err.Error())
		return
	}
	if !handleDestroyPlanResponseDiagnostics(resp, rsp.JSON400, rsp.JSON401, rsp.JSON403, rsp.JSON404, rsp.JSON409, rsp.JSON500, rsp.Status()) {
		return
	}
	if rsp.JSON202 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 202 response when creating destroy plan for job %q in app %q in org %q, got %s.", data.JobSlug.ValueString(), data.AppSlug.ValueString(), data.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	plan := rsp.JSON202.Data
	reportDestroyPlanProgress(resp, "Destroy plan", plan)
	if data.Apply.ValueBool() {
		applyDestroyPlan(ctx, a.client, data.OrgName.ValueString(), plan, resp)
	}
}
