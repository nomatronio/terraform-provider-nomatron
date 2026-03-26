package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &RoleDataSource{}

type RoleDataSource struct {
	client *sdk.ClientWithResponses
}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

type RoleDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.List   `tfsdk:"permissions"`
	BuiltIn     types.Bool   `tfsdk:"built_in"`
	Scope       types.String `tfsdk:"scope"`
}

func (d *RoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron role by exact name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Role ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact role name.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Role description.",
			},
			"permissions": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Permission strings assigned to the role.",
			},
			"built_in": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the role is built in.",
			},
			"scope": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Role scope selector.",
			},
		},
	}
}

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the role data source.",
		)
		return
	}

	role, err := readRoleByName(ctx, d.client, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Role", err.Error())
		return
	}

	data, diags := flattenRoleDataSource(role)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *RoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func readRoleByName(ctx context.Context, client *sdk.ClientWithResponses, name string) (sdk.RoleDetail, error) {
	rsp, err := client.GetRoleWithResponse(ctx, name)
	if err != nil {
		return sdk.RoleDetail{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.RoleDetail{}, fmt.Errorf("no role was found with name %q", name)
	}

	if rsp.JSON200 == nil {
		return sdk.RoleDetail{}, fmt.Errorf("expected 200 response when reading role %q, got %s", name, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func flattenRoleDataSource(role sdk.RoleDetail) (RoleDataSourceModel, diag.Diagnostics) {
	description := types.StringNull()
	if role.Description != nil {
		description = types.StringValue(*role.Description)
	}

	permissions, diags := types.ListValueFrom(context.Background(), types.StringType, role.Permissions)
	if diags.HasError() {
		return RoleDataSourceModel{}, diags
	}

	scope := types.StringNull()
	if role.Scope != "" {
		scope = types.StringValue(role.Scope)
	}

	return RoleDataSourceModel{
		ID:          types.StringValue(role.Id.String()),
		Name:        types.StringValue(role.Name),
		Description: description,
		Permissions: permissions,
		BuiltIn:     types.BoolValue(role.BuiltIn),
		Scope:       scope,
	}, nil
}
