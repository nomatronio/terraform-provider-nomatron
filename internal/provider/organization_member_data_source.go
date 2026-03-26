package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &OrganizationMemberDataSource{}

type OrganizationMemberDataSource struct {
	client *sdk.ClientWithResponses
}

func NewOrganizationMemberDataSource() datasource.DataSource {
	return &OrganizationMemberDataSource{}
}

type OrganizationMemberDataSourceModel struct {
	OrgName  types.String `tfsdk:"org_name"`
	Username types.String `tfsdk:"username"`
	UserID   types.String `tfsdk:"user_id"`
	JoinedAt types.String `tfsdk:"joined_at"`
}

func (d *OrganizationMemberDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member"
}

func (d *OrganizationMemberDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron organization member by organization name and username.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name.",
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Username to look up within the organization.",
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User ID for the organization member.",
			},
			"joined_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the user joined the organization.",
			},
		},
	}
}

func (d *OrganizationMemberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationMemberDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization member data source.",
		)
		return
	}

	member, err := readOrganizationMember(ctx, d.client, data.OrgName.ValueString(), data.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Organization Member", err.Error())
		return
	}

	data = flattenOrganizationMemberDataSource(data, member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *OrganizationMemberDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenOrganizationMemberDataSource(base OrganizationMemberDataSourceModel, member sdk.OrganizationMember) OrganizationMemberDataSourceModel {
	userID := types.StringNull()
	if member.UserId.String() != "00000000-0000-0000-0000-000000000000" {
		userID = types.StringValue(member.UserId.String())
	}

	joinedAt := types.StringNull()
	if !member.JoinedAt.IsZero() {
		joinedAt = types.StringValue(member.JoinedAt.Format(time.RFC3339))
	}

	username := base.Username
	if member.Username != "" {
		username = types.StringValue(member.Username)
	}

	return OrganizationMemberDataSourceModel{
		OrgName:  base.OrgName,
		Username: username,
		UserID:   userID,
		JoinedAt: joinedAt,
	}
}
