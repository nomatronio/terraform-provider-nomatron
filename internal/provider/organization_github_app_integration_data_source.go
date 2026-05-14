package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &OrganizationGitHubAppIntegrationDataSource{}

type OrganizationGitHubAppIntegrationDataSource struct {
	client *sdk.ClientWithResponses
}

func NewOrganizationGitHubAppIntegrationDataSource() datasource.DataSource {
	return &OrganizationGitHubAppIntegrationDataSource{}
}

type OrganizationGitHubAppIntegrationDataSourceModel struct {
	OrgName       types.String `tfsdk:"org_name"`
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

func (d *OrganizationGitHubAppIntegrationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_github_app_integration"
}

func (d *OrganizationGitHubAppIntegrationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single organization-scoped Nomatron GitHub App integration by organization name and exact integration name.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the integration.",
			},
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
				MarkdownDescription: "Integration scope, typically `org` for this data source.",
			},
		},
	}
}

func (d *OrganizationGitHubAppIntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationGitHubAppIntegrationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization github app integration data source.",
		)
		return
	}

	connection, err := readOrganizationGitHubIntegration(ctx, d.client, data.OrgName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Organization GitHub App Integration", err.Error())
		return
	}

	data = flattenOrganizationGitHubAppIntegrationDataSource(data, connection)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *OrganizationGitHubAppIntegrationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenOrganizationGitHubAppIntegrationDataSource(base OrganizationGitHubAppIntegrationDataSourceModel, connection sdk.GitHubConnection) OrganizationGitHubAppIntegrationDataSourceModel {
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

	return OrganizationGitHubAppIntegrationDataSourceModel{
		OrgName:       base.OrgName,
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
