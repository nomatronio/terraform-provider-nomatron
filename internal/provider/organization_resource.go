package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &OrganizationResource{}
var _ resource.ResourceWithImportState = &OrganizationResource{}

type OrganizationResource struct {
	client *sdk.ClientWithResponses
}

func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

type OrganizationResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Metadata      types.Map    `tfsdk:"metadata"`
	OwnerUsername types.String `tfsdk:"owner_username"`
	OwnerUserID   types.String `tfsdk:"owner_user_id"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *OrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron organization resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Organization ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Organization description.",
			},
			"metadata": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Metadata attached to the organization.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"owner_username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional owner username to set during creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owner_user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Owner user ID.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the organization was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the organization was last updated.",
			},
		},
	}
}

func (r *OrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization resource.",
		)
		return
	}

	body := sdk.CreateOrganizationJSONRequestBody{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}

	if !plan.OwnerUsername.IsNull() && !plan.OwnerUsername.IsUnknown() {
		ownerUsername := plan.OwnerUsername.ValueString()
		body.OwnerUsername = &ownerUsername
	}

	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		md, diags := terraformMapToStringMap(ctx, plan.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(md) > 0 {
			metadata := mapStringToAny(md)
			body.Metadata = &metadata
		}
	}

	rsp, err := r.client.CreateOrganizationWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Organization", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Organization", formatAPIError(rsp.JSON400))
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
	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Organization Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating organization, got %s.", rsp.Status()),
		)
		return
	}

	state, diags := stateFromOrganization(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization resource.",
		)
		return
	}

	rsp, err := r.client.GetOrganizationWithResponse(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Organization", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		resp.State.RemoveResource(ctx)
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
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when reading organization %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}

	newState, diags := stateFromOrganization(state, rsp.JSON200.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationResourceModel
	var state OrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization resource.",
		)
		return
	}

	body := sdk.UpdateOrganizationJSONRequestBody{}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}

	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		md, diags := terraformMapToStringMap(ctx, plan.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(md) > 0 {
			var metadataUnion sdk.UpdateOrganizationRequest_Metadata
			if err := metadataUnion.FromUpdateOrganizationRequestMetadata0(sdk.UpdateOrganizationRequestMetadata0(mapStringToAny(md))); err != nil {
				resp.Diagnostics.AddError("Invalid Metadata", err.Error())
				return
			}
			body.Metadata = &metadataUnion
		}
	}

	rsp, err := r.client.UpdateOrganizationWithResponse(ctx, state.Name.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Organization", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Organization", formatAPIError(rsp.JSON400))
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
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating organization %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}

	newState, diags := stateFromOrganization(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization resource.",
		)
		return
	}

	rsp, err := r.client.DeleteOrganizationWithResponse(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Organization", err.Error())
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
	if rsp.StatusCode() != http.StatusOK && rsp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting organization %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func stateFromOrganization(base OrganizationResourceModel, org sdk.Organization) (OrganizationResourceModel, diag.Diagnostics) {
	metadata := base.Metadata
	if metadata.IsUnknown() {
		metadata = types.MapNull(types.StringType)
	}
	if org.Metadata != nil {
		stringMetadata := make(map[string]string, len(*org.Metadata))
		for k, v := range *org.Metadata {
			stringMetadata[k] = fmt.Sprintf("%v", v)
		}

		mv, diags := types.MapValueFrom(context.Background(), types.StringType, stringMetadata)
		if diags.HasError() {
			return OrganizationResourceModel{}, diags
		}
		metadata = mv
	}

	description := base.Description
	if description.IsUnknown() {
		description = types.StringNull()
	}
	if org.Description != nil {
		description = types.StringValue(*org.Description)
	}

	return OrganizationResourceModel{
		ID:            types.StringValue(org.Id.String()),
		Name:          types.StringValue(org.Name),
		Description:   description,
		Metadata:      metadata,
		OwnerUsername: base.OwnerUsername,
		OwnerUserID:   types.StringValue(org.OwnerUserId.String()),
		CreatedAt:     types.StringValue(org.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:     types.StringValue(org.UpdatedAt.Format(time.RFC3339)),
	}, nil
}
