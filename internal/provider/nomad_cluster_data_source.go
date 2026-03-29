package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &NomadClusterDataSource{}

type NomadClusterDataSource struct {
	client *sdk.ClientWithResponses
}

func NewNomadClusterDataSource() datasource.DataSource {
	return &NomadClusterDataSource{}
}

type NomadClusterDataSourceModel struct {
	Name             types.String `tfsdk:"name"`
	ID               types.String `tfsdk:"id"`
	Description      types.String `tfsdk:"description"`
	ConnectivityMode types.String `tfsdk:"connectivity_mode"`
	Address          types.String `tfsdk:"address"`
	AgentID          types.String `tfsdk:"agent_id"`
	SkipVerify       types.Bool   `tfsdk:"skip_verify"`
	Scope            types.String `tfsdk:"scope"`
}

func (d *NomadClusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nomad_cluster"
}

func (d *NomadClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single global Nomatron Nomad cluster by exact name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact global cluster name.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster ID.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster description.",
			},
			"connectivity_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Connectivity mode, either `direct` or `agent`.",
			},
			"address": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Nomad API address for `direct` connectivity mode.",
			},
			"agent_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Agent ID for `agent` connectivity mode.",
			},
			"skip_verify": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether TLS verification is skipped for direct connections.",
			},
			"scope": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster scope, typically `global` for this data source.",
			},
		},
	}
}

func (d *NomadClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NomadClusterDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the nomad cluster data source.",
		)
		return
	}

	cluster, err := readGlobalCluster(ctx, d.client, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Nomad Cluster", err.Error())
		return
	}

	data = flattenNomadClusterDataSource(data, cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *NomadClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenNomadClusterDataSource(base NomadClusterDataSourceModel, cluster sdk.Cluster) NomadClusterDataSourceModel {
	description := types.StringNull()
	if cluster.Description != nil {
		description = types.StringValue(*cluster.Description)
	}

	address := types.StringNull()
	if cluster.Address != nil {
		address = types.StringValue(*cluster.Address)
	}

	agentID := types.StringNull()
	if cluster.NetworkAgentId != nil {
		agentID = types.StringValue(cluster.NetworkAgentId.String())
	}

	return NomadClusterDataSourceModel{
		Name:             base.Name,
		ID:               types.StringValue(cluster.Id.String()),
		Description:      description,
		ConnectivityMode: types.StringValue(string(cluster.ConnectivityMode)),
		Address:          address,
		AgentID:          agentID,
		SkipVerify:       types.BoolValue(cluster.SkipVerify),
		Scope:            types.StringValue(cluster.Scope),
	}
}
