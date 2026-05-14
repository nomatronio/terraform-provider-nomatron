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

var _ resource.Resource = &GroupResource{}
var _ resource.ResourceWithImportState = &GroupResource{}

type GroupResource struct {
	client *sdk.ClientWithResponses
}

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

type GroupResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrgName        types.String `tfsdk:"org_name"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Metadata       types.Map    `tfsdk:"metadata"`
	OwnerUsername  types.String `tfsdk:"owner_username"`
	OrganizationID types.String `tfsdk:"organization_id"`
	OwnerUserID    types.String `tfsdk:"owner_user_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *GroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron organization-scoped group resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Group ID.",
			},
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Group name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Group description.",
			},
			"metadata": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Metadata attached to the group.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"owner_username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Username that will own the group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Organization ID that owns the group.",
			},
			"owner_user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Owner user ID.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the group was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the group was last updated.",
			},
		},
	}
}

func (r *GroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group resource.",
		)
		return
	}

	body := sdk.PostOrganizationsOrgSlugGroupsJSONRequestBody{
		Name:          plan.Name.ValueString(),
		OwnerUsername: plan.OwnerUsername.ValueString(),
	}

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
			metadata := mapStringToAny(md)
			body.Metadata = &metadata
		}
	}

	rsp, err := r.client.PostOrganizationsOrgSlugGroupsWithResponse(ctx, plan.OrgName.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Group", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Group", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Organization Not Found", formatAPIError(rsp.JSON404))
		return
	}
	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Group Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Create Group", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating group %q in organization %q, got %s.", plan.Name.ValueString(), plan.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	state, diags := stateFromGroup(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group resource.",
		)
		return
	}

	group, err := readGroup(ctx, r.client, state.OrgName.ValueString(), state.Name.ValueString())
	if err != nil {
		if isGroupNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Group", err.Error())
		return
	}

	newState, diags := stateFromGroup(state, group)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupResourceModel
	var state GroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group resource.",
		)
		return
	}

	body := sdk.PatchOrganizationsOrgSlugGroupsGroupNameJSONRequestBody{}

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
			metadata := mapStringToAny(md)
			body.Metadata = &metadata
		}
	}

	rsp, err := r.client.PatchOrganizationsOrgSlugGroupsGroupNameWithResponse(ctx, state.OrgName.ValueString(), state.Name.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Group", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Group", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Update Group", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating group %q in organization %q, got %s.", state.Name.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	newState, diags := stateFromGroup(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group resource.",
		)
		return
	}

	rsp, err := r.client.DeleteOrganizationsOrgSlugGroupsGroupNameWithResponse(ctx, state.OrgName.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Group", err.Error())
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
		resp.Diagnostics.AddError("Failed To Delete Group", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting group %q in organization %q, got %s.", state.Name.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgName, groupName, err := parseGroupImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), groupName)...)
}

type groupNotFoundError struct {
	orgName string
	name    string
}

func (e *groupNotFoundError) Error() string {
	return fmt.Sprintf("group %q in organization %q not found", e.name, e.orgName)
}

func isGroupNotFound(err error) bool {
	_, ok := err.(*groupNotFoundError)
	return ok
}

func readGroup(ctx context.Context, client *sdk.ClientWithResponses, orgName, groupName string) (sdk.Group, error) {
	rsp, err := client.GetOrganizationsOrgSlugGroupsGroupNameWithResponse(ctx, orgName, groupName)
	if err != nil {
		return sdk.Group{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.Group{}, &groupNotFoundError{orgName: orgName, name: groupName}
	}
	if rsp.JSON200 == nil {
		return sdk.Group{}, fmt.Errorf("expected 200 response when reading group %q in organization %q, got %s", groupName, orgName, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func stateFromGroup(base GroupResourceModel, group sdk.Group) (GroupResourceModel, diag.Diagnostics) {
	description := base.Description
	if description.IsUnknown() {
		description = types.StringNull()
	}
	if group.Description != nil {
		description = types.StringValue(*group.Description)
	}

	metadata := base.Metadata
	if metadata.IsUnknown() {
		metadata = types.MapNull(types.StringType)
	}
	if group.Metadata != nil {
		stringMetadata := make(map[string]string, len(*group.Metadata))
		for k, v := range *group.Metadata {
			stringMetadata[k] = fmt.Sprintf("%v", v)
		}

		mv, diags := types.MapValueFrom(context.Background(), types.StringType, stringMetadata)
		if diags.HasError() {
			return GroupResourceModel{}, diags
		}
		metadata = mv
	}

	return GroupResourceModel{
		ID:             types.StringValue(group.Id.String()),
		OrgName:        base.OrgName,
		Name:           types.StringValue(group.Name),
		Description:    description,
		Metadata:       metadata,
		OwnerUsername:  base.OwnerUsername,
		OrganizationID: types.StringValue(group.OrganizationId.String()),
		OwnerUserID:    types.StringValue(group.OwnerUserId.String()),
		CreatedAt:      types.StringValue(group.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:      types.StringValue(group.UpdatedAt.Format(time.RFC3339)),
	}, nil
}

func parseGroupImportID(raw string) (orgName, groupName string, err error) {
	parts := pathExpressionSplit(raw)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid import id %q: expected org_name/group_name", raw)
	}

	return parts[0], parts[1], nil
}

func pathExpressionSplit(raw string) []string {
	var parts []string
	current := ""
	for _, r := range raw {
		if r == '/' {
			parts = append(parts, current)
			current = ""
			continue
		}
		current += string(r)
	}
	parts = append(parts, current)
	return parts
}
