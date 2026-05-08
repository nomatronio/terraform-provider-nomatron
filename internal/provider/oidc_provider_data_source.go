package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &OIDCProviderDataSource{}

type OIDCProviderDataSource struct {
	client *sdk.ClientWithResponses
}

func NewOIDCProviderDataSource() datasource.DataSource {
	return &OIDCProviderDataSource{}
}

type OIDCProviderDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Slug                types.String `tfsdk:"slug"`
	DisplayName         types.String `tfsdk:"display_name"`
	IssuerURL           types.String `tfsdk:"issuer_url"`
	ClientID            types.String `tfsdk:"client_id"`
	Scopes              types.List   `tfsdk:"scopes"`
	UsernameClaim       types.String `tfsdk:"username_claim"`
	EmailClaim          types.String `tfsdk:"email_claim"`
	NameClaim           types.String `tfsdk:"name_claim"`
	GroupsClaim         types.String `tfsdk:"groups_claim"`
	AllowedEmailDomains types.List   `tfsdk:"allowed_email_domains"`
	AutoProvision       types.Bool   `tfsdk:"auto_provision"`
	SyncProfile         types.Bool   `tfsdk:"sync_profile"`
	SyncGroups          types.Bool   `tfsdk:"sync_groups"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	IsDefault           types.Bool   `tfsdk:"is_default"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func (d *OIDCProviderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_provider"
}

func (d *OIDCProviderDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron OIDC provider by slug.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OIDC provider ID.",
			},
			"slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OIDC provider slug.",
			},
			"display_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Human-readable provider name.",
			},
			"issuer_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OIDC issuer URL.",
			},
			"client_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OIDC client ID.",
			},
			"scopes": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "OIDC scopes requested during login.",
			},
			"username_claim": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Claim used for the Nomatron username.",
			},
			"email_claim": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Claim used for the user's email address.",
			},
			"name_claim": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Claim used for the display name.",
			},
			"groups_claim": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Claim containing external group values.",
			},
			"allowed_email_domains": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Email domain allow-list.",
			},
			"auto_provision": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether valid first-time OIDC logins auto-create Nomatron users.",
			},
			"sync_profile": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether profile fields are synchronized at login.",
			},
			"sync_groups": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether external group claims synchronize mapped RBAC assignments at login.",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this provider can be used for login.",
			},
			"is_default": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this provider is the default OIDC login provider.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the provider was last updated.",
			},
		},
	}
}

func (d *OIDCProviderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OIDCProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OIDCProviderDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC provider data source.")
		return
	}

	provider, err := readOIDCProvider(ctx, d.client, data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read OIDC Provider", err.Error())
		return
	}

	state, diags := stateFromOIDCProvider(OIDCProviderResourceModel{}, provider)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = OIDCProviderDataSourceModel{
		ID:                  state.ID,
		Slug:                state.Slug,
		DisplayName:         state.DisplayName,
		IssuerURL:           state.IssuerURL,
		ClientID:            state.ClientID,
		Scopes:              state.Scopes,
		UsernameClaim:       state.UsernameClaim,
		EmailClaim:          state.EmailClaim,
		NameClaim:           state.NameClaim,
		GroupsClaim:         state.GroupsClaim,
		AllowedEmailDomains: state.AllowedEmailDomains,
		AutoProvision:       state.AutoProvision,
		SyncProfile:         state.SyncProfile,
		SyncGroups:          state.SyncGroups,
		Enabled:             state.Enabled,
		IsDefault:           state.IsDefault,
		UpdatedAt:           state.UpdatedAt,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
