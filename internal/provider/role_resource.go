package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}

type RoleResource struct {
	client *sdk.ClientWithResponses
}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

type RoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.List   `tfsdk:"permissions"`
	BuiltIn     types.Bool   `tfsdk:"built_in"`
	Scope       types.String `tfsdk:"scope"`
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron role resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Role ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Role name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Role description.",
			},
			"permissions": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Permission strings assigned to the role.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"built_in": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the role is built in.",
			},
			"scope": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Role scope selector.",
			},
		},
	}
}

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the role resource.",
		)
		return
	}

	permissions, diags := terraformListToStringSlice(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.CreateRoleJSONRequestBody{
		Name:        plan.Name.ValueString(),
		Permissions: permissions,
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}

	rsp, err := r.client.CreateRoleWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Role", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Role", formatAPIError(rsp.JSON400))
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
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating role, got %s.", rsp.Status()),
		)
		return
	}

	role, err := r.readRole(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Role", err.Error())
		return
	}

	state, diags := stateFromRole(plan, role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the role resource.",
		)
		return
	}

	role, err := r.readRole(ctx, state.Name.ValueString())
	if err != nil {
		if isRoleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Role", err.Error())
		return
	}

	newState, diags := stateFromRole(state, role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleResourceModel
	var state RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the role resource.",
		)
		return
	}

	body := sdk.UpdateRoleJSONRequestBody{}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}

	permissions, diags := terraformListToStringSlice(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	body.Permissions = &permissions

	rsp, err := r.client.UpdateRoleWithResponse(ctx, state.Name.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Role", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Role", formatAPIError(rsp.JSON400))
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
			fmt.Sprintf("Expected 200 response when updating role %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}

	role, err := r.readRole(ctx, state.Name.ValueString())
	if err != nil {
		if isRoleNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Role", err.Error())
		return
	}

	newState, diags := stateFromRole(plan, role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the role resource.",
		)
		return
	}

	rsp, err := r.client.DeleteRoleWithResponse(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Role", err.Error())
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
			fmt.Sprintf("Expected 200/204 response when deleting role %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

type roleNotFoundError struct {
	name string
}

func (e *roleNotFoundError) Error() string {
	return fmt.Sprintf("role %q not found", e.name)
}

func isRoleNotFound(err error) bool {
	_, ok := err.(*roleNotFoundError)
	return ok
}

func (r *RoleResource) readRole(ctx context.Context, name string) (sdk.RoleDetail, error) {
	rsp, err := r.client.GetRoleWithResponse(ctx, name)
	if err != nil {
		return sdk.RoleDetail{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.RoleDetail{}, &roleNotFoundError{name: name}
	}
	if rsp.JSON401 != nil {
		return sdk.RoleDetail{}, fmt.Errorf("unauthorized: %s", formatAPIError(rsp.JSON401))
	}
	if rsp.JSON403 != nil {
		return sdk.RoleDetail{}, fmt.Errorf("forbidden: %s", formatAPIError(rsp.JSON403))
	}
	if rsp.JSON200 == nil {
		return sdk.RoleDetail{}, fmt.Errorf("expected 200 response when reading role %q, got %s", name, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func stateFromRole(base RoleResourceModel, role sdk.RoleDetail) (RoleResourceModel, diag.Diagnostics) {
	permissions, diags := types.ListValueFrom(context.Background(), types.StringType, role.Permissions)
	if diags.HasError() {
		return RoleResourceModel{}, diags
	}

	description := base.Description
	if description.IsUnknown() {
		description = types.StringNull()
	}
	if role.Description != nil {
		description = types.StringValue(*role.Description)
	}

	scope := types.StringNull()
	if role.Scope != "" {
		scope = types.StringValue(role.Scope)
	}

	return RoleResourceModel{
		ID:          types.StringValue(role.Id.String()),
		Name:        types.StringValue(role.Name),
		Description: description,
		Permissions: permissions,
		BuiltIn:     types.BoolValue(role.BuiltIn),
		Scope:       scope,
	}, nil
}

func terraformListToStringSlice(ctx context.Context, value types.List) ([]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	out := make([]string, 0)
	diags := value.ElementsAs(ctx, &out, false)
	return out, diags
}
