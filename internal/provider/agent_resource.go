package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
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

var _ resource.Resource = &AgentResource{}
var _ resource.ResourceWithImportState = &AgentResource{}

type AgentResource struct {
	client *sdk.ClientWithResponses
}

func NewAgentResource() resource.Resource {
	return &AgentResource{}
}

type AgentResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	IsActive      types.Bool   `tfsdk:"is_active"`
	CreatedAt     types.String `tfsdk:"created_at"`
	CreatedByType types.String `tfsdk:"created_by_type"`
	CreatedByID   types.String `tfsdk:"created_by_id"`
}

func (r *AgentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connectivity_agent"
}

func (r *AgentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron connectivity agent resource.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Agent ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the connectivity agent.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description for the connectivity agent.",
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the connectivity agent is active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the connectivity agent was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Type of actor that created the connectivity agent.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the actor that created the connectivity agent, when available.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AgentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the connectivity agent resource.",
		)
		return
	}

	body := sdk.CreateConnectivityAgentJSONRequestBody{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}

	rsp, err := r.client.CreateConnectivityAgentWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Connectivity Agent", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Connectivity Agent", formatErrorEnvelope(rsp.JSON400))
		return
	}

	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatErrorEnvelope(rsp.JSON401))
		return
	}

	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Connectivity Agent Already Exists", formatErrorEnvelope(rsp.JSON409))
		return
	}

	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating connectivity agent, got %s.", rsp.Status()),
		)
		return
	}

	agent := rsp.JSON201.Data.ConnectivityAgent
	agentID := agent.Id
	if agentID == uuid.Nil {
		agentID = rsp.JSON201.Data.Token.ConnectivityAgentId
	}
	if agentID == uuid.Nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			"Create connectivity agent response did not include an agent ID.",
		)
		return
	}

	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() && !plan.IsActive.ValueBool() && agent.IsActive {
		deactivateResp, err := r.client.DeactivateConnectivityAgentWithResponse(ctx, agentID)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Deactivate Connectivity Agent", err.Error())
			return
		}

		if deactivateResp.JSON400 != nil {
			resp.Diagnostics.AddError("Failed To Deactivate Connectivity Agent", formatErrorEnvelope(deactivateResp.JSON400))
			return
		}

		if deactivateResp.JSON401 != nil {
			resp.Diagnostics.AddError("Unauthorized", formatErrorEnvelope(deactivateResp.JSON401))
			return
		}

		if deactivateResp.JSON404 != nil {
			resp.Diagnostics.AddError("Connectivity Agent Not Found", formatErrorEnvelope(deactivateResp.JSON404))
			return
		}

		if deactivateResp.JSON200 == nil {
			resp.Diagnostics.AddError(
				"Unexpected API Response",
				fmt.Sprintf("Expected 200 response when deactivating connectivity agent %q after create, got %s.", agent.Id.String(), deactivateResp.Status()),
			)
			return
		}

		agent = deactivateResp.JSON200.Data.ConnectivityAgent
		if agent.Id != uuid.Nil {
			agentID = agent.Id
		}
	}

	agent, err = r.readAgent(ctx, agentID)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Connectivity Agent", err.Error())
		return
	}
	if agent.Id == uuid.Nil {
		agent.Id = agentID
	}

	state := stateFromAgent(plan, agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the connectivity agent resource.",
		)
		return
	}

	agentID, err := parseAgentID(state.ID.ValueString())
	if err != nil {
		agentID, err = r.resolveAgentID(ctx, state)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Connectivity Agent ID", err.Error())
			return
		}
	}

	agent, err := r.readAgent(ctx, agentID)
	if err != nil {
		var notFoundErr *agentNotFoundError
		if ok := errorAs(err, &notFoundErr); ok {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Failed To Read Connectivity Agent", err.Error())
		return
	}

	newState := stateFromAgent(state, agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *AgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AgentResourceModel
	var state AgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the connectivity agent resource.",
		)
		return
	}

	agentID, err := parseAgentID(state.ID.ValueString())
	if err != nil {
		agentID, err = r.resolveAgentID(ctx, state)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Connectivity Agent ID", err.Error())
			return
		}
	}

	body := sdk.UpdateConnectivityAgentJSONRequestBody{}
	shouldPatch := false

	if stringValueChanged(plan.Name, state.Name) && !plan.Name.IsUnknown() && !plan.Name.IsNull() {
		name := plan.Name.ValueString()
		body.Name = &name
		shouldPatch = true
	}

	if stringValueChanged(plan.Description, state.Description) && !plan.Description.IsUnknown() && !plan.Description.IsNull() {
		description := plan.Description.ValueString()
		body.Description = &description
		shouldPatch = true
	}

	currentAgent := sdk.ConnectivityAgent{}
	if shouldPatch {
		rsp, err := r.client.UpdateConnectivityAgentWithResponse(ctx, agentID, body)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Update Connectivity Agent", err.Error())
			return
		}

		if rsp.JSON400 != nil {
			resp.Diagnostics.AddError("Failed To Update Connectivity Agent", formatErrorEnvelope(rsp.JSON400))
			return
		}

		if rsp.JSON401 != nil {
			resp.Diagnostics.AddError("Unauthorized", formatErrorEnvelope(rsp.JSON401))
			return
		}

		if rsp.JSON404 != nil {
			resp.State.RemoveResource(ctx)
			return
		}

		if rsp.JSON200 == nil {
			resp.Diagnostics.AddError(
				"Unexpected API Response",
				fmt.Sprintf("Expected 200 response when updating connectivity agent %q, got %s.", state.ID.ValueString(), rsp.Status()),
			)
			return
		}

		currentAgent = rsp.JSON200.Data.ConnectivityAgent
	}

	if boolValueChanged(plan.IsActive, state.IsActive) && !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() {
		if plan.IsActive.ValueBool() {
			activateResp, err := r.client.ActivateConnectivityAgentWithResponse(ctx, agentID, sdk.ActivateConnectivityAgentJSONRequestBody{})
			if err != nil {
				resp.Diagnostics.AddError("Failed To Activate Connectivity Agent", err.Error())
				return
			}

			if activateResp.JSON400 != nil {
				resp.Diagnostics.AddError("Failed To Activate Connectivity Agent", formatErrorEnvelope(activateResp.JSON400))
				return
			}

			if activateResp.JSON401 != nil {
				resp.Diagnostics.AddError("Unauthorized", formatErrorEnvelope(activateResp.JSON401))
				return
			}

			if activateResp.JSON404 != nil {
				resp.State.RemoveResource(ctx)
				return
			}

			if activateResp.JSON201 == nil {
				resp.Diagnostics.AddError(
					"Unexpected API Response",
					fmt.Sprintf("Expected 201 response when activating connectivity agent %q, got %s.", state.ID.ValueString(), activateResp.Status()),
				)
				return
			}

			currentAgent = activateResp.JSON201.Data.ConnectivityAgent
		} else {
			deactivateResp, err := r.client.DeactivateConnectivityAgentWithResponse(ctx, agentID)
			if err != nil {
				resp.Diagnostics.AddError("Failed To Deactivate Connectivity Agent", err.Error())
				return
			}

			if deactivateResp.JSON400 != nil {
				resp.Diagnostics.AddError("Failed To Deactivate Connectivity Agent", formatErrorEnvelope(deactivateResp.JSON400))
				return
			}

			if deactivateResp.JSON401 != nil {
				resp.Diagnostics.AddError("Unauthorized", formatErrorEnvelope(deactivateResp.JSON401))
				return
			}

			if deactivateResp.JSON404 != nil {
				resp.State.RemoveResource(ctx)
				return
			}

			if deactivateResp.JSON200 == nil {
				resp.Diagnostics.AddError(
					"Unexpected API Response",
					fmt.Sprintf("Expected 200 response when deactivating connectivity agent %q, got %s.", state.ID.ValueString(), deactivateResp.Status()),
				)
				return
			}

			currentAgent = deactivateResp.JSON200.Data.ConnectivityAgent
		}
	}

	if currentAgent.Id == uuid.Nil {
		currentAgent, err = r.readAgent(ctx, agentID)
		if err != nil {
			var notFoundErr *agentNotFoundError
			if ok := errorAs(err, &notFoundErr); ok {
				resp.State.RemoveResource(ctx)
				return
			}

			resp.Diagnostics.AddError("Failed To Read Connectivity Agent", err.Error())
			return
		}
	} else {
		currentAgent, err = r.readAgent(ctx, currentAgent.Id)
		if err != nil {
			var notFoundErr *agentNotFoundError
			if ok := errorAs(err, &notFoundErr); ok {
				resp.State.RemoveResource(ctx)
				return
			}

			resp.Diagnostics.AddError("Failed To Read Connectivity Agent", err.Error())
			return
		}
	}

	newState := stateFromAgent(plan, currentAgent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *AgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the connectivity agent resource.",
		)
		return
	}

	agentID, err := parseAgentID(state.ID.ValueString())
	if err != nil {
		agentID, err = r.resolveAgentID(ctx, state)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Connectivity Agent ID", err.Error())
			return
		}
	}

	rsp, err := r.client.DeleteConnectivityAgentWithResponse(ctx, agentID)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Connectivity Agent", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Delete Connectivity Agent", formatErrorEnvelope(rsp.JSON400))
		return
	}

	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatErrorEnvelope(rsp.JSON401))
		return
	}

	if rsp.StatusCode() != http.StatusOK && rsp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting connectivity agent %q, got %s.", state.ID.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *AgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func stateFromAgent(base AgentResourceModel, agent sdk.ConnectivityAgent) AgentResourceModel {
	id := base.ID
	if agent.Id != uuid.Nil {
		id = types.StringValue(agent.Id.String())
	}

	name := base.Name
	if name.IsUnknown() {
		name = types.StringNull()
	}
	if agent.Name != "" {
		name = types.StringValue(agent.Name)
	}

	description := base.Description
	if description.IsUnknown() {
		description = types.StringNull()
	}
	if agent.Description == nil {
	} else {
		description = types.StringValue(*agent.Description)
	}

	isActive := base.IsActive
	if isActive.IsUnknown() || isActive.IsNull() || agent.Id != uuid.Nil {
		isActive = types.BoolValue(agent.IsActive)
	}

	createdAt := base.CreatedAt
	if createdAt.IsUnknown() {
		createdAt = types.StringNull()
	}
	if !agent.CreatedAt.IsZero() {
		createdAt = types.StringValue(agent.CreatedAt.Format(time.RFC3339))
	}

	createdByType := types.StringNull()
	if agent.CreatedByType != "" {
		createdByType = types.StringValue(string(agent.CreatedByType))
	}

	createdByID := types.StringNull()
	if flattened := flattenAgentCreatedByID(agent); flattened != "" {
		createdByID = types.StringValue(flattened)
	}

	return AgentResourceModel{
		ID:            id,
		Name:          name,
		Description:   description,
		IsActive:      isActive,
		CreatedAt:     createdAt,
		CreatedByType: createdByType,
		CreatedByID:   createdByID,
	}
}

func flattenAgentCreatedByID(agent sdk.ConnectivityAgent) string {
	if agent.CreatedByUserId != nil {
		return agent.CreatedByUserId.String()
	}
	if agent.CreatedByServiceAccountId != nil {
		return agent.CreatedByServiceAccountId.String()
	}
	return ""
}

func parseAgentID(raw string) (openapi_types.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return openapi_types.UUID{}, fmt.Errorf("invalid UUID %q: %w", raw, err)
	}

	return openapi_types.UUID(id), nil
}

func (r *AgentResource) resolveAgentID(ctx context.Context, state AgentResourceModel) (openapi_types.UUID, error) {
	if !state.ID.IsNull() && !state.ID.IsUnknown() && state.ID.ValueString() != "" {
		return parseAgentID(state.ID.ValueString())
	}

	if state.Name.IsNull() || state.Name.IsUnknown() || state.Name.ValueString() == "" {
		return openapi_types.UUID{}, fmt.Errorf("connectivity agent id is missing from state and name is unavailable for lookup")
	}

	query := state.Name.ValueString()
	params := &sdk.ListConnectivityAgentsParams{
		Q: &query,
	}

	rsp, err := r.client.ListConnectivityAgentsWithResponse(ctx, params)
	if err != nil {
		return openapi_types.UUID{}, err
	}

	if rsp.JSON200 == nil {
		return openapi_types.UUID{}, fmt.Errorf("expected 200 response when listing connectivity agents for %q, got %s", query, rsp.Status())
	}

	var matches []sdk.ConnectivityAgent
	for _, agent := range rsp.JSON200.Data.Items {
		if agent.Name == query {
			matches = append(matches, agent)
		}
	}

	switch len(matches) {
	case 0:
		return openapi_types.UUID{}, fmt.Errorf("connectivity agent id is missing from state and no agent named %q was found", query)
	case 1:
		return matches[0].Id, nil
	default:
		return openapi_types.UUID{}, fmt.Errorf("connectivity agent id is missing from state and multiple agents named %q were found", query)
	}
}

type agentNotFoundError struct {
	id string
}

func (e *agentNotFoundError) Error() string {
	return fmt.Sprintf("connectivity agent %q not found", e.id)
}

func (r *AgentResource) readAgent(ctx context.Context, agentID openapi_types.UUID) (sdk.ConnectivityAgent, error) {
	rsp, err := r.client.ReadConnectivityAgentWithResponse(ctx, agentID)
	if err != nil {
		return sdk.ConnectivityAgent{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.ConnectivityAgent{}, &agentNotFoundError{id: agentID.String()}
	}

	if rsp.JSON200 == nil {
		return sdk.ConnectivityAgent{}, fmt.Errorf("expected 200 response when reading connectivity agent %q, got %s", agentID.String(), rsp.Status())
	}

	return rsp.JSON200.Data.ConnectivityAgent, nil
}

func errorAs(err error, target interface{}) bool {
	switch t := target.(type) {
	case **agentNotFoundError:
		notFoundErr, ok := err.(*agentNotFoundError)
		if ok {
			*t = notFoundErr
			return true
		}
	}

	return false
}

func formatErrorEnvelope(apiErr *sdk.ErrorEnvelope) string {
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

func stringValueChanged(a, b types.String) bool {
	switch {
	case a.IsNull() && b.IsNull():
		return false
	case a.IsNull() != b.IsNull():
		return true
	case a.IsUnknown() || b.IsUnknown():
		return false
	default:
		return a.ValueString() != b.ValueString()
	}
}

func boolValueChanged(a, b types.Bool) bool {
	switch {
	case a.IsNull() && b.IsNull():
		return false
	case a.IsNull() != b.IsNull():
		return true
	case a.IsUnknown() || b.IsUnknown():
		return false
	default:
		return a.ValueBool() != b.ValueBool()
	}
}
