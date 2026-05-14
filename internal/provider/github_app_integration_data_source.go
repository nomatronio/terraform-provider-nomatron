package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &GitHubAppIntegrationDataSource{}

type GitHubAppIntegrationDataSource struct {
	client *sdk.ClientWithResponses
}

func NewGitHubAppIntegrationDataSource() datasource.DataSource {
	return &GitHubAppIntegrationDataSource{}
}

type GitHubAppIntegrationDataSourceModel struct {
	Name          types.String `tfsdk:"name"`
	ID            types.String `tfsdk:"id"`
	ProviderKind  types.String `tfsdk:"provider_kind"`
	WebBaseURL    types.String `tfsdk:"web_base_url"`
	APIBaseURL    types.String `tfsdk:"api_base_url"`
	UploadBaseURL types.String `tfsdk:"upload_base_url"`
	AppID         types.String `tfsdk:"app_id"`
	AppSlug       types.String `tfsdk:"app_slug"`
	ClientID      types.String `tfsdk:"client_id"`
	Scope         types.String `tfsdk:"scope"`
}

func (d *GitHubAppIntegrationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app_integration"
}

func (d *GitHubAppIntegrationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single global Nomatron GitHub App integration by exact name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact integration name.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Integration ID.",
			},
			"provider_kind": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "GitHub provider kind.",
			},
			"web_base_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "GitHub web base URL.",
			},
			"api_base_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "GitHub API base URL.",
			},
			"upload_base_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "GitHub upload API base URL.",
			},
			"app_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "GitHub App ID.",
			},
			"app_slug": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "GitHub App slug.",
			},
			"client_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "GitHub App client ID.",
			},
			"scope": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Integration scope, typically `global` for this data source.",
			},
		},
	}
}

func (d *GitHubAppIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GitHubAppIntegrationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the github app integration data source.",
		)
		return
	}

	connection, err := readGlobalGitHubIntegration(ctx, d.client, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read GitHub App Integration", err.Error())
		return
	}

	data = flattenGitHubAppIntegrationDataSource(data, connection)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *GitHubAppIntegrationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenGitHubAppIntegrationDataSource(base GitHubAppIntegrationDataSourceModel, connection sdk.GitHubConnection) GitHubAppIntegrationDataSourceModel {
	appID := types.StringNull()
	if connection.AppId != nil {
		appID = types.StringValue(*connection.AppId)
	}

	appSlug := types.StringNull()
	if connection.AppSlug != nil {
		appSlug = types.StringValue(*connection.AppSlug)
	}

	clientID := types.StringNull()
	if connection.ClientId != nil {
		clientID = types.StringValue(*connection.ClientId)
	}
	providerKind := types.StringNull()
	if connection.ProviderKind != nil {
		providerKind = types.StringValue(string(*connection.ProviderKind))
	}
	webBaseURL := types.StringNull()
	if connection.WebBaseUrl != nil {
		webBaseURL = types.StringValue(*connection.WebBaseUrl)
	}
	apiBaseURL := types.StringNull()
	if connection.ApiBaseUrl != nil {
		apiBaseURL = types.StringValue(*connection.ApiBaseUrl)
	}
	uploadBaseURL := types.StringNull()
	if connection.UploadBaseUrl != nil {
		uploadBaseURL = types.StringValue(*connection.UploadBaseUrl)
	}

	return GitHubAppIntegrationDataSourceModel{
		Name:          base.Name,
		ID:            types.StringValue(connection.Id.String()),
		ProviderKind:  providerKind,
		WebBaseURL:    webBaseURL,
		APIBaseURL:    apiBaseURL,
		UploadBaseURL: uploadBaseURL,
		AppID:         appID,
		AppSlug:       appSlug,
		ClientID:      clientID,
		Scope:         types.StringValue(string(connection.Scope)),
	}
}
