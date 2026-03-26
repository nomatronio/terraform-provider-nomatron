package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ action.Action = &SpeculativePlanAction{}
var _ action.ActionWithConfigure = &SpeculativePlanAction{}

type SpeculativePlanAction struct {
	client *sdk.ClientWithResponses
}

func NewSpeculativePlanAction() action.Action {
	return &SpeculativePlanAction{}
}

type SpeculativePlanActionModel struct {
	OrgName types.String `tfsdk:"org_name"`
	AppSlug types.String `tfsdk:"app_slug"`
	JobSlug types.String `tfsdk:"job_slug"`
	Ref     types.String `tfsdk:"ref"`
}

func (a *SpeculativePlanAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_speculative_plan"
}

func (a *SpeculativePlanAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Trigger a speculative plan for an existing Nomatron job.",
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
				MarkdownDescription: "Nomatron job slug to plan.",
			},
			"ref": actionschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional git ref override for the speculative plan.",
			},
		},
	}
}

func (a *SpeculativePlanAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *SpeculativePlanAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data SpeculativePlanActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if a.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the speculative plan action.",
		)
		return
	}

	if resp.SendProgress != nil {
		message := fmt.Sprintf("Triggering speculative plan for job %q.", data.JobSlug.ValueString())
		if !data.Ref.IsNull() && !data.Ref.IsUnknown() && data.Ref.ValueString() != "" {
			message = fmt.Sprintf("Triggering speculative plan for job %q at ref %q.", data.JobSlug.ValueString(), data.Ref.ValueString())
		}
		resp.SendProgress(action.InvokeProgressEvent{Message: message})
	}

	body := sdk.CreateSpeculativePlanJSONRequestBody{}
	if !data.Ref.IsNull() && !data.Ref.IsUnknown() && data.Ref.ValueString() != "" {
		ref := data.Ref.ValueString()
		body.Ref = &ref
	}

	rsp, err := a.client.CreateSpeculativePlanWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Trigger Speculative Plan", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Trigger Speculative Plan", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Job Not Found", formatAPIError(rsp.JSON404))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Trigger Speculative Plan", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when creating speculative plan for job %q in app %q in org %q, got %s.", data.JobSlug.ValueString(), data.AppSlug.ValueString(), data.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	message := fmt.Sprintf("Speculative plan %q is %q.", rsp.JSON200.Data.PlanId.String(), rsp.JSON200.Data.Status)
	if rsp.JSON200.Data.Summary != nil && *rsp.JSON200.Data.Summary != "" {
		message = fmt.Sprintf("%s Summary: %s", message, *rsp.JSON200.Data.Summary)
	}
	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{Message: message})
	}
}
