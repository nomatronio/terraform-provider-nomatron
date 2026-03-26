// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

// Ensure TerraCurlProvider satisfies various provider interfaces.
var _ provider.Provider = &NomatronProvider{}
var _ provider.ProviderWithFunctions = &NomatronProvider{}
var _ provider.ProviderWithActions = &NomatronProvider{}

// NomatronProvider defines the provider implementation.
type NomatronProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NomatronProviderModel describes the provider data model.
type NomatronProviderModel struct {
	Address   types.String `tfsdk:"address"`
	Token     types.String `tfsdk:"token"`
	TlsCert   types.String `tfsdk:"tls_cert"`
	TlsKey    types.String `tfsdk:"tls_key"`
	TlsCaCert types.String `tfsdk:"tls_ca_cert"`
}

func (p *NomatronProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nomatron"
	resp.Version = p.version
}

func (p *NomatronProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	//resp.Schema = schema.Schema{
	//	MarkdownDescription: "The Nomatron provider allows you to configure Nomatron clusters and its resources.",
	//}
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				MarkdownDescription: "The Nomatron API address.",
				Required:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The Nomatron service account token.",
				Required:            true,
			},
			"tls_cert": schema.StringAttribute{
				MarkdownDescription: "The path to the TLS certificate file.",
				Optional:            true,
			},
			"tls_key": schema.StringAttribute{
				MarkdownDescription: "The path to the TLS key file.",
				Optional:            true,
			},
			"tls_ca_cert": schema.StringAttribute{
				MarkdownDescription: "The path to the TLS CA certificate file.",
				Optional:            true,
			},
		},
	}
}

func (p *NomatronProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data NomatronProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := configureClient(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
	resp.EphemeralResourceData = client
	resp.ActionData = client
}

func configureClient(data NomatronProviderModel) (*sdk.ClientWithResponses, error) {
	useTLS := !data.TlsCert.IsNull() || !data.TlsKey.IsNull() || !data.TlsCaCert.IsNull()

	address := normalizeAddress(data.Address.ValueString(), useTLS)

	httpClient, err := buildHTTPClient(data)
	if err != nil {
		return nil, err
	}

	return sdk.NewClientWithResponses(
		address,
		sdk.WithHTTPClient(httpClient),
		sdk.WithBearerToken(data.Token.ValueString()),
	)
}

func (p *NomatronProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAgentResource,
		NewApplicationResource,
		NewEnvironmentResource,
		NewGitHubAppIntegrationResource,
		NewOrganizationGitHubAppIntegrationResource,
		NewJobResource,
		NewNomadClusterResource,
		NewOrganizationNomadClusterResource,
		NewGroupMemberResource,
		NewGroupResource,
		NewOrganizationMemberResource,
		NewOrganizationResource,
		NewRoleResource,
		NewRoleAssignmentResource,
		NewServiceAccountResource,
		NewVariableResource,
		NewUserResource,
	}
}

func (p *NomatronProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAgentDataSource,
		NewApplicationDataSource,
		NewEnvironmentDataSource,
		NewGitHubAppIntegrationDataSource,
		NewOrganizationGitHubAppIntegrationDataSource,
		NewJobDataSource,
		NewNomadClusterDataSource,
		NewOrganizationNomadClusterDataSource,
		NewGroupMemberDataSource,
		NewGroupDataSource,
		NewOrganizationMemberDataSource,
		NewOrganizationDataSource,
		NewRoleDataSource,
		NewServiceAccountDataSource,
		NewUsersDataSource,
	}
}

func (p *NomatronProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewAgentTokenEphemeralResource,
		NewServiceAccountTokenEphemeralResource,
	}
}

func (p *NomatronProvider) Actions(ctx context.Context) []func() action.Action {
	return []func() action.Action{
		NewImportNomadJobAction,
		NewSpeculativePlanAction,
	}
}

func (p *NomatronProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NomatronProvider{
			version: version,
		}
	}
}
