package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &OrganizationNomadClusterDataSource{}

type OrganizationNomadClusterDataSource struct {
	client *sdk.ClientWithResponses
}

func NewOrganizationNomadClusterDataSource() datasource.DataSource {
	return &OrganizationNomadClusterDataSource{}
}

type OrganizationNomadClusterDataSourceModel struct {
	OrgName          types.String `tfsdk:"org_name"`
	Name             types.String `tfsdk:"name"`
	ID               types.String `tfsdk:"id"`
	Description      types.String `tfsdk:"description"`
	ConnectivityMode types.String `tfsdk:"connectivity_mode"`
	Address          types.String `tfsdk:"address"`
	AgentID          types.String `tfsdk:"agent_id"`
	SkipVerify       types.Bool   `tfsdk:"skip_verify"`
	Scope            types.String `tfsdk:"scope"`
}

func (d *OrganizationNomadClusterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_nomad_cluster"
}

func (d *OrganizationNomadClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single organization-scoped Nomatron Nomad cluster by organization name and exact cluster name.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the cluster.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact organization cluster name.",
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
				MarkdownDescription: "Cluster scope, typically the owning organization ID for this data source.",
			},
		},
	}
}

func (d *OrganizationNomadClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationNomadClusterDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization nomad cluster data source.",
		)
		return
	}

	cluster, err := readOrganizationCluster(ctx, d.client, data.OrgName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Organization Nomad Cluster", err.Error())
		return
	}

	data = flattenOrganizationNomadClusterDataSource(data, cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *OrganizationNomadClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenOrganizationNomadClusterDataSource(base OrganizationNomadClusterDataSourceModel, cluster sdk.Cluster) OrganizationNomadClusterDataSourceModel {
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

	return OrganizationNomadClusterDataSourceModel{
		OrgName:          base.OrgName,
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
