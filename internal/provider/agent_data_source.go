package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ datasource.DataSource = &AgentDataSource{}

type AgentDataSource struct {
	client *sdk.ClientWithResponses
}

func NewAgentDataSource() datasource.DataSource {
	return &AgentDataSource{}
}

type AgentDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	IsActive      types.Bool   `tfsdk:"is_active"`
	CreatedAt     types.String `tfsdk:"created_at"`
	CreatedByType types.String `tfsdk:"created_by_type"`
	CreatedByID   types.String `tfsdk:"created_by_id"`
}

func (d *AgentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (d *AgentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron agent by id or exact name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Agent ID. If omitted, the provider can look up the agent by exact name.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Exact agent name. If omitted, the provider reads the agent by id.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description for the agent.",
			},
			"is_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the agent is active.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the agent was created.",
			},
			"created_by_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Type of actor that created the agent.",
			},
			"created_by_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of the actor that created the agent, when available.",
			},
		},
	}
}

func (d *AgentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AgentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the agent data source.",
		)
		return
	}

	var (
		agent sdk.Agent
		err   error
	)

	if !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != "" {
		agentID, parseErr := parseAgentID(data.ID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid Agent ID", parseErr.Error())
			return
		}

		agent, err = readAgentByID(ctx, d.client, agentID)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Read Agent", err.Error())
			return
		}
	} else if !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != "" {
		agent, err = findAgentByExactName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed To Read Agent", err.Error())
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Lookup Attribute",
			"Set either id or name to read a single agent.",
		)
		return
	}

	data = flattenAgentDataSource(agent)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *AgentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func flattenAgentDataSource(agent sdk.Agent) AgentDataSourceModel {
	description := types.StringNull()
	if agent.Description != nil {
		description = types.StringValue(*agent.Description)
	}

	createdByType := types.StringNull()
	if agent.CreatedByType != "" {
		createdByType = types.StringValue(string(agent.CreatedByType))
	}

	createdByID := types.StringNull()
	if flattened := flattenAgentCreatedByID(agent); flattened != "" {
		createdByID = types.StringValue(flattened)
	}

	createdAt := types.StringNull()
	if !agent.CreatedAt.IsZero() {
		createdAt = types.StringValue(agent.CreatedAt.Format(time.RFC3339))
	}

	return AgentDataSourceModel{
		ID:            types.StringValue(agent.Id.String()),
		Name:          types.StringValue(agent.Name),
		Description:   description,
		IsActive:      types.BoolValue(agent.IsActive),
		CreatedAt:     createdAt,
		CreatedByType: createdByType,
		CreatedByID:   createdByID,
	}
}

func readAgentByID(ctx context.Context, client *sdk.ClientWithResponses, agentID openapi_types.UUID) (sdk.Agent, error) {
	rsp, err := client.ReadAgentWithResponse(ctx, agentID)
	if err != nil {
		return sdk.Agent{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.Agent{}, fmt.Errorf("no agent was found with id %q", agentID.String())
	}

	if rsp.JSON200 == nil {
		return sdk.Agent{}, fmt.Errorf("expected 200 response when reading agent %q, got %s", agentID.String(), rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func findAgentByExactName(ctx context.Context, client *sdk.ClientWithResponses, name string) (sdk.Agent, error) {
	params := &sdk.ListAgentsParams{
		Q: &name,
	}

	rsp, err := client.ListAgentsWithResponse(ctx, params)
	if err != nil {
		return sdk.Agent{}, err
	}

	if rsp.JSON200 == nil {
		return sdk.Agent{}, fmt.Errorf("expected 200 response when listing agents for %q, got %s", name, rsp.Status())
	}

	var matches []sdk.Agent
	for _, agent := range rsp.JSON200.Data.Items {
		if agent.Name == name {
			matches = append(matches, agent)
		}
	}

	switch len(matches) {
	case 0:
		return sdk.Agent{}, fmt.Errorf("no agent was found with name %q", name)
	case 1:
		return matches[0], nil
	default:
		return sdk.Agent{}, fmt.Errorf("multiple agents were found with name %q", name)
	}
}
