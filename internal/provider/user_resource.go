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

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

type UserResource struct {
	client *sdk.ClientWithResponses
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Username     types.String `tfsdk:"username"`
	Name         types.String `tfsdk:"name"`
	Password     types.String `tfsdk:"password"`
	Metadata     types.Map    `tfsdk:"metadata"`
	AuthProvider types.String `tfsdk:"auth_provider"`
	IsActive     types.Bool   `tfsdk:"is_active"`
	CreatedAt    types.String `tfsdk:"created_at"`
	CreatedBy    types.String `tfsdk:"created_by"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron user resource.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User ID.",
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Username for the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "Initial password for the user.",
			},
			"metadata": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Metadata attached to the user.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the user.",
			},

			"auth_provider": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Authentication provider for the user.",
			},
			"is_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the user is active.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the user was created.",
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Actor that created the user.",
			},
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel
	var config UserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the user resource.",
		)
		return
	}

	body := sdk.CreateUserJSONRequestBody{
		Username: plan.Username.ValueString(),
		Name:     plan.Name.ValueString(),
		Password: config.Password.ValueString(),
	}

	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		md, diags := terraformMapToStringMap(ctx, plan.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(md) > 0 {
			var metadataUnion sdk.CreateUserRequest_Metadata
			if err := metadataUnion.FromCreateUserRequestMetadata0(mapStringToAny(md)); err != nil {
				resp.Diagnostics.AddError("Invalid Metadata", err.Error())
				return
			}
			body.Metadata = &metadataUnion
		}
	}

	rsp, err := r.client.CreateUserWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create User", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError(
			"Failed To Create User",
			formatAPIError(rsp.JSON400),
		)
		return
	}

	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError(
			"Unauthorized",
			formatAPIError(rsp.JSON401),
		)
		return
	}

	if rsp.JSON403 != nil {
		resp.Diagnostics.AddError(
			"Forbidden",
			formatAPIError(rsp.JSON403),
		)
		return
	}

	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError(
			"User Already Exists",
			formatAPIError(rsp.JSON409),
		)
		return
	}

	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating user, got %s.", rsp.Status()),
		)
		return
	}

	state, diags := stateFromUser(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the user resource.",
		)
		return
	}

	rsp, err := r.client.GetUserWithResponse(ctx, state.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read User", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when reading user %q, got %s.", state.Username.ValueString(), rsp.Status()),
		)
		return
	}

	newState, diags := stateFromUser(state, rsp.JSON200.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserResourceModel
	var state UserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the user resource.",
		)
		return
	}

	body := sdk.UpdateUserJSONRequestBody{}

	if !plan.Name.IsUnknown() && !plan.Name.IsNull() {
		name := plan.Name.ValueString()
		body.Name = &name
	}

	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		md, diags := terraformMapToStringMap(ctx, plan.Metadata)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		var metadataUnion sdk.UpdateUserRequest_Metadata
		if err := metadataUnion.FromUpdateUserRequestMetadata0(mapStringToAny(md)); err != nil {
			resp.Diagnostics.AddError("Invalid Metadata", err.Error())
			return
		}
		body.Metadata = &metadataUnion
	}

	rsp, err := r.client.UpdateUserWithResponse(ctx, state.Username.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update User", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating user %q, got %s.", state.Username.ValueString(), rsp.Status()),
		)
		return
	}

	newState, diags := stateFromUser(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the user resource.",
		)
		return
	}

	rsp, err := r.client.DeleteUserWithResponse(ctx, state.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete User", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		return
	}

	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting user %q, got %s.", state.Username.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("username"), req, resp)
}

func stateFromUser(base UserResourceModel, u sdk.User) (UserResourceModel, diag.Diagnostics) {
	metadata := base.Metadata
	if metadata.IsUnknown() {
		metadata = types.MapNull(types.StringType)
	}
	if u.Metadata != nil {
		mv, diags := types.MapValueFrom(context.Background(), types.StringType, *u.Metadata)
		if diags.HasError() {
			return UserResourceModel{}, diags
		}
		metadata = mv
	}

	return UserResourceModel{
		ID:           types.StringValue(u.Id.String()),
		Username:     types.StringValue(u.Username),
		Name:         types.StringValue(u.Name),
		Password:     base.Password,
		Metadata:     metadata,
		AuthProvider: types.StringValue(u.AuthProvider),
		IsActive:     types.BoolValue(u.IsActive),
		CreatedAt:    types.StringValue(u.CreatedAt.Format(time.RFC3339)),
		CreatedBy:    types.StringValue(flattenCreatedBy(u.CreatedBy)),
	}, nil
}

func terraformMapToStringMap(ctx context.Context, value types.Map) (map[string]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}

	out := make(map[string]string)
	diags := value.ElementsAs(ctx, &out, false)
	return out, diags
}

func mapStringToAny(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func formatAPIError(apiErr *sdk.ApiResponseError) string {
	if apiErr == nil {
		return "unknown API error"
	}

	if len(apiErr.Errors) == 0 {
		return fmt.Sprintf("%s (%s)", apiErr.Meta.Code, apiErr.Meta.Status)
	}

	msg := apiErr.Errors[0].Message
	if apiErr.Errors[0].Field != nil && *apiErr.Errors[0].Field != "" {
		return fmt.Sprintf("%s: %s", *apiErr.Errors[0].Field, msg)
	}

	return msg
}
