package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ action.Action = &DestroyAppJobsAction{}
var _ action.ActionWithConfigure = &DestroyAppJobsAction{}

type DestroyAppJobsAction struct {
	client *sdk.ClientWithResponses
}

func NewDestroyAppJobsAction() action.Action {
	return &DestroyAppJobsAction{}
}

type DestroyAppJobsActionModel struct {
	OrgName types.String `tfsdk:"org_name"`
	AppSlug types.String `tfsdk:"app_slug"`
	Apply   types.Bool   `tfsdk:"apply"`
}

func (a *DestroyAppJobsAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destroy_app_jobs"
}

func (a *DestroyAppJobsAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Create destroy plans for all live Nomad jobs in one Nomatron application. This stops and purges runtime state only; it does not delete Nomatron application or job resources.",
		Attributes: map[string]actionschema.Attribute{
			"org_name": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the application.",
			},
			"app_slug": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application slug whose live Nomad jobs should be stopped and purged.",
			},
			"apply": actionschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When true, immediately queue destroy applies for ready destroy plans. Approval-required plans still need approval before apply can be queued.",
			},
		},
	}
}

func (a *DestroyAppJobsAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *DestroyAppJobsAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data DestroyAppJobsActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if a.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the destroy app jobs action.",
		)
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: fmt.Sprintf("Creating app-level destroy plans for app %q.", data.AppSlug.ValueString()),
		})
	}

	rsp, err := a.client.CreateAppDestroyPlanWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create App Destroy Plans", err.Error())
		return
	}
	if !handleDestroyPlanResponseDiagnostics(resp, rsp.JSON400, rsp.JSON401, rsp.JSON403, rsp.JSON404, rsp.JSON409, rsp.JSON500, rsp.Status()) {
		return
	}
	if rsp.JSON202 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 202 response when creating app destroy plans for app %q in org %q, got %s.", data.AppSlug.ValueString(), data.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	plan := rsp.JSON202.Data
	reportDestroyPlanProgress(resp, "App destroy plan", plan)
	if data.Apply.ValueBool() {
		applyDestroyPlan(ctx, a.client, data.OrgName.ValueString(), plan, resp)
	}
}
