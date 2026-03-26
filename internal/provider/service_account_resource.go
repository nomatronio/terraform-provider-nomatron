package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ resource.Resource = &ServiceAccountResource{}
var _ resource.ResourceWithImportState = &ServiceAccountResource{}

type ServiceAccountResource struct {
	client *sdk.ClientWithResponses
}

func NewServiceAccountResource() resource.Resource {
	return &ServiceAccountResource{}
}

type ServiceAccountResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	CreatedAt   types.String `tfsdk:"created_at"`
	CreatedBy   types.String `tfsdk:"created_by"`
}

func (r *ServiceAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *ServiceAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron service account resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Service account ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Service account name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Service account description.",
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the service account is active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the service account was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Actor ID that created the service account.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ServiceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServiceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the service account resource.",
		)
		return
	}

	body := sdk.CreateServiceAccountJSONRequestBody{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}

	rsp, err := r.client.CreateServiceAccountWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Service Account", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Service Account", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Service Account Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating service account, got %s.", rsp.Status()),
		)
		return
	}

	account := rsp.JSON201.Data

	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() && !plan.IsActive.ValueBool() && account.IsActive {
		deactivateResp, err := r.client.DeactivateServiceAccountWithResponse(ctx, account.Id)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Deactivate Service Account", err.Error())
			return
		}

		if deactivateResp.JSON400 != nil {
			resp.Diagnostics.AddError("Failed To Deactivate Service Account", formatAPIError(deactivateResp.JSON400))
			return
		}
		if deactivateResp.JSON401 != nil {
			resp.Diagnostics.AddError("Unauthorized", formatAPIError(deactivateResp.JSON401))
			return
		}
		if deactivateResp.JSON403 != nil {
			resp.Diagnostics.AddError("Forbidden", formatAPIError(deactivateResp.JSON403))
			return
		}
		if deactivateResp.JSON404 != nil {
			resp.Diagnostics.AddError("Service Account Not Found", formatAPIError(deactivateResp.JSON404))
			return
		}
		if deactivateResp.JSON200 == nil {
			resp.Diagnostics.AddError(
				"Unexpected API Response",
				fmt.Sprintf("Expected 200 response when deactivating service account %q, got %s.", account.Id.String(), deactivateResp.Status()),
			)
			return
		}

		account = deactivateResp.JSON200.Data
	}

	state := stateFromServiceAccount(plan, account)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the service account resource.",
		)
		return
	}

	accountID, err := parseServiceAccountID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Service Account ID", err.Error())
		return
	}

	account, err := r.readServiceAccount(ctx, accountID)
	if err != nil {
		if isServiceAccountNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Service Account", err.Error())
		return
	}

	newState := stateFromServiceAccount(state, account)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ServiceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ServiceAccountResourceModel
	var state ServiceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the service account resource.",
		)
		return
	}

	accountID, err := parseServiceAccountID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Service Account ID", err.Error())
		return
	}

	body := sdk.UpdateServiceAccountJSONRequestBody{}

	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		name := plan.Name.ValueString()
		body.Name = &name
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}

	rsp, err := r.client.UpdateServiceAccountWithResponse(ctx, accountID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Service Account", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Service Account", formatAPIError(rsp.JSON400))
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
	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Service Account Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating service account %q, got %s.", state.ID.ValueString(), rsp.Status()),
		)
		return
	}

	account := rsp.JSON200.Data

	if boolValueChanged(plan.IsActive, state.IsActive) && !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() {
		if plan.IsActive.ValueBool() {
			activateResp, err := r.client.ActivateServiceAccountWithResponse(ctx, accountID)
			if err != nil {
				resp.Diagnostics.AddError("Failed To Activate Service Account", err.Error())
				return
			}

			if activateResp.JSON400 != nil {
				resp.Diagnostics.AddError("Failed To Activate Service Account", formatAPIError(activateResp.JSON400))
				return
			}
			if activateResp.JSON401 != nil {
				resp.Diagnostics.AddError("Unauthorized", formatAPIError(activateResp.JSON401))
				return
			}
			if activateResp.JSON403 != nil {
				resp.Diagnostics.AddError("Forbidden", formatAPIError(activateResp.JSON403))
				return
			}
			if activateResp.JSON404 != nil {
				resp.State.RemoveResource(ctx)
				return
			}
			if activateResp.JSON200 == nil {
				resp.Diagnostics.AddError(
					"Unexpected API Response",
					fmt.Sprintf("Expected 200 response when activating service account %q, got %s.", state.ID.ValueString(), activateResp.Status()),
				)
				return
			}

			account = activateResp.JSON200.Data
		} else {
			deactivateResp, err := r.client.DeactivateServiceAccountWithResponse(ctx, accountID)
			if err != nil {
				resp.Diagnostics.AddError("Failed To Deactivate Service Account", err.Error())
				return
			}

			if deactivateResp.JSON400 != nil {
				resp.Diagnostics.AddError("Failed To Deactivate Service Account", formatAPIError(deactivateResp.JSON400))
				return
			}
			if deactivateResp.JSON401 != nil {
				resp.Diagnostics.AddError("Unauthorized", formatAPIError(deactivateResp.JSON401))
				return
			}
			if deactivateResp.JSON403 != nil {
				resp.Diagnostics.AddError("Forbidden", formatAPIError(deactivateResp.JSON403))
				return
			}
			if deactivateResp.JSON404 != nil {
				resp.State.RemoveResource(ctx)
				return
			}
			if deactivateResp.JSON200 == nil {
				resp.Diagnostics.AddError(
					"Unexpected API Response",
					fmt.Sprintf("Expected 200 response when deactivating service account %q, got %s.", state.ID.ValueString(), deactivateResp.Status()),
				)
				return
			}

			account = deactivateResp.JSON200.Data
		}
	}

	newState := stateFromServiceAccount(plan, account)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ServiceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the service account resource.",
		)
		return
	}

	accountID, err := parseServiceAccountID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Service Account ID", err.Error())
		return
	}

	rsp, err := r.client.DeleteServiceAccountWithResponse(ctx, accountID)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Service Account", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		return
	}
	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Delete Service Account", formatAPIError(rsp.JSON400))
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
			fmt.Sprintf("Expected 200/204 response when deleting service account %q, got %s.", state.ID.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *ServiceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type serviceAccountNotFoundError struct {
	id string
}

func (e *serviceAccountNotFoundError) Error() string {
	return fmt.Sprintf("service account %q not found", e.id)
}

func isServiceAccountNotFound(err error) bool {
	_, ok := err.(*serviceAccountNotFoundError)
	return ok
}

func parseServiceAccountID(raw string) (openapi_types.UUID, error) {
	return parseAgentID(raw)
}

func (r *ServiceAccountResource) readServiceAccount(ctx context.Context, accountID openapi_types.UUID) (sdk.ServiceAccount, error) {
	rsp, err := r.client.ReadServiceAccountWithResponse(ctx, accountID)
	if err != nil {
		return sdk.ServiceAccount{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.ServiceAccount{}, &serviceAccountNotFoundError{id: accountID.String()}
	}
	if rsp.JSON401 != nil {
		return sdk.ServiceAccount{}, fmt.Errorf("unauthorized: %s", formatAPIError(rsp.JSON401))
	}
	if rsp.JSON403 != nil {
		return sdk.ServiceAccount{}, fmt.Errorf("forbidden: %s", formatAPIError(rsp.JSON403))
	}
	if rsp.JSON200 == nil {
		return sdk.ServiceAccount{}, fmt.Errorf("expected 200 response when reading service account %q, got %s", accountID.String(), rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func stateFromServiceAccount(base ServiceAccountResourceModel, account sdk.ServiceAccount) ServiceAccountResourceModel {
	zeroUUID := openapi_types.UUID{}

	id := base.ID
	if account.Id != zeroUUID {
		id = types.StringValue(account.Id.String())
	}

	name := base.Name
	if name.IsUnknown() {
		name = types.StringNull()
	}
	if account.Name != "" {
		name = types.StringValue(account.Name)
	}

	description := base.Description
	if description.IsUnknown() {
		description = types.StringNull()
	}
	if account.Description != nil {
		description = types.StringValue(*account.Description)
	}

	isActive := base.IsActive
	if isActive.IsUnknown() || isActive.IsNull() || account.Id != zeroUUID {
		isActive = types.BoolValue(account.IsActive)
	}

	createdAt := base.CreatedAt
	if createdAt.IsUnknown() {
		createdAt = types.StringNull()
	}
	if !account.CreatedAt.IsZero() {
		createdAt = types.StringValue(account.CreatedAt.Format(time.RFC3339))
	}

	createdBy := base.CreatedBy
	if createdBy.IsUnknown() {
		createdBy = types.StringNull()
	}
	if account.CreatedBy != zeroUUID {
		createdBy = types.StringValue(account.CreatedBy.String())
	}

	return ServiceAccountResourceModel{
		ID:          id,
		Name:        name,
		Description: description,
		IsActive:    isActive,
		CreatedAt:   createdAt,
		CreatedBy:   createdBy,
	}
}
