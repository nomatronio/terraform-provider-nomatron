package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &OIDCGroupMappingDataSource{}

type OIDCGroupMappingDataSource struct {
	client *sdk.ClientWithResponses
}

func NewOIDCGroupMappingDataSource() datasource.DataSource {
	return &OIDCGroupMappingDataSource{}
}

type OIDCGroupMappingDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProviderSlug  types.String `tfsdk:"provider_slug"`
	ProviderID    types.String `tfsdk:"provider_id"`
	ExternalGroup types.String `tfsdk:"external_group"`
	Role          types.String `tfsdk:"role"`
	Domain        types.String `tfsdk:"domain"`
	Description   types.String `tfsdk:"description"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (d *OIDCGroupMappingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_group_mapping"
}

func (d *OIDCGroupMappingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron OIDC group mapping by id or exact provider_slug/external_group/role/domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "OIDC group mapping ID. If omitted, the provider looks up the mapping by provider_slug, external_group, role, and domain.",
			},
			"provider_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OIDC provider slug that owns this mapping.",
			},
			"provider_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OIDC provider ID.",
			},
			"external_group": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "External OIDC group claim value.",
			},
			"role": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Nomatron role name assigned to matching users.",
			},
			"domain": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "RBAC domain, for example global or an organization domain.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mapping description.",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this mapping is active.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the mapping was created.",
			},
		},
	}
}

func (d *OIDCGroupMappingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OIDCGroupMappingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OIDCGroupMappingDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC group mapping data source.")
		return
	}

	var (
		mapping sdk.OIDCGroupRoleMapping
		err     error
	)

	if !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != "" {
		mappingID, parseErr := parseAgentID(data.ID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid OIDC Group Mapping ID", parseErr.Error())
			return
		}
		mapping, err = readOIDCGroupMapping(ctx, d.client, data.ProviderSlug.ValueString(), mappingID)
	} else {
		if data.ExternalGroup.IsNull() || data.ExternalGroup.IsUnknown() || data.ExternalGroup.ValueString() == "" ||
			data.Role.IsNull() || data.Role.IsUnknown() || data.Role.ValueString() == "" ||
			data.Domain.IsNull() || data.Domain.IsUnknown() || data.Domain.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Lookup Attributes",
				"Set id, or set external_group, role, and domain to read a single OIDC group mapping.",
			)
			return
		}
		mapping, err = findOIDCGroupMapping(ctx, d.client, data.ProviderSlug.ValueString(), data.ExternalGroup.ValueString(), data.Role.ValueString(), data.Domain.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError("Failed To Read OIDC Group Mapping", err.Error())
		return
	}

	state := stateFromOIDCGroupMapping(data.ProviderSlug.ValueString(), mapping)
	data = OIDCGroupMappingDataSourceModel{
		ID:            state.ID,
		ProviderSlug:  state.ProviderSlug,
		ProviderID:    state.ProviderID,
		ExternalGroup: state.ExternalGroup,
		Role:          state.Role,
		Domain:        state.Domain,
		Description:   state.Description,
		Enabled:       state.Enabled,
		CreatedAt:     state.CreatedAt,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
