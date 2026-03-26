package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	actionschema "github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ action.Action = &ImportNomadJobAction{}
var _ action.ActionWithConfigure = &ImportNomadJobAction{}

type ImportNomadJobAction struct {
	client *sdk.ClientWithResponses
}

func NewImportNomadJobAction() action.Action {
	return &ImportNomadJobAction{}
}

type ImportNomadJobActionModel struct {
	OrgName types.String `tfsdk:"org_name"`
	AppSlug types.String `tfsdk:"app_slug"`
	JobSlug types.String `tfsdk:"job_slug"`
	JobID   types.String `tfsdk:"job_id"`
}

func (a *ImportNomadJobAction) Metadata(ctx context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_import_nomad_job"
}

func (a *ImportNomadJobAction) Schema(ctx context.Context, req action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = actionschema.Schema{
		MarkdownDescription: "Queue a Nomad job import for an existing Nomatron job.",
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
				MarkdownDescription: "Nomatron job slug to import into.",
			},
			"job_id": actionschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Nomad job ID to import.",
			},
		},
	}
}

func (a *ImportNomadJobAction) Configure(ctx context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *ImportNomadJobAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var data ImportNomadJobActionModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if a.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the import Nomad job action.",
		)
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: fmt.Sprintf("Queueing Nomad job import for %q into job %q.", data.JobID.ValueString(), data.JobSlug.ValueString()),
		})
	}

	body := sdk.ImportJobJSONRequestBody{
		JobId: data.JobID.ValueString(),
	}

	rsp, err := a.client.ImportJobWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Import Nomad Job", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Import Nomad Job", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Import Nomad Job", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON202 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 202 response when importing Nomad job %q into job %q for app %q in org %q, got %s.", data.JobID.ValueString(), data.JobSlug.ValueString(), data.AppSlug.ValueString(), data.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	if resp.SendProgress != nil {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: fmt.Sprintf("Queued Nomad job import operation %q with status %q.", rsp.JSON202.Data.OperationId.String(), rsp.JSON202.Data.Status),
		})
	}
}
