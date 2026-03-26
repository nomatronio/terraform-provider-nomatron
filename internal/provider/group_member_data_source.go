package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &GroupMemberDataSource{}

type GroupMemberDataSource struct {
	client *sdk.ClientWithResponses
}

func NewGroupMemberDataSource() datasource.DataSource {
	return &GroupMemberDataSource{}
}

type GroupMemberDataSourceModel struct {
	OrgName   types.String `tfsdk:"org_name"`
	GroupName types.String `tfsdk:"group_name"`
	Username  types.String `tfsdk:"username"`
}

func (d *GroupMemberDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_member"
}

func (d *GroupMemberDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron group member by organization name, group name, and username. Note: the current group-member endpoints expose membership presence by username only.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the group.",
			},
			"group_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Group name.",
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Username to look up in the group.",
			},
		},
	}
}

func (d *GroupMemberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupMemberDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group member data source.",
		)
		return
	}

	exists, err := groupMemberExists(ctx, d.client, data.OrgName.ValueString(), data.GroupName.ValueString(), data.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Group Member", err.Error())
		return
	}
	if !exists {
		resp.Diagnostics.AddError(
			"Group Member Not Found",
			fmt.Sprintf("No group member %q was found in group %q for organization %q.", data.Username.ValueString(), data.GroupName.ValueString(), data.OrgName.ValueString()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *GroupMemberDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
