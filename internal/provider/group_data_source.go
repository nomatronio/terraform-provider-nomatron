package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &GroupDataSource{}

type GroupDataSource struct {
	client *sdk.ClientWithResponses
}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

type GroupDataSourceModel struct {
	OrgName        types.String `tfsdk:"org_name"`
	Name           types.String `tfsdk:"name"`
	ID             types.String `tfsdk:"id"`
	Description    types.String `tfsdk:"description"`
	Metadata       types.Map    `tfsdk:"metadata"`
	OrganizationID types.String `tfsdk:"organization_id"`
	OwnerUserID    types.String `tfsdk:"owner_user_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *GroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron group by organization name and exact group name.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the group.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact group name.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Group ID.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Group description.",
			},
			"metadata": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Metadata attached to the group.",
			},
			"organization_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Organization ID that owns the group.",
			},
			"owner_user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Owner user ID.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the group was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the group was last updated.",
			},
		},
	}
}

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group data source.",
		)
		return
	}

	group, err := readGroup(ctx, d.client, data.OrgName.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Group", err.Error())
		return
	}

	data, diags := flattenGroupDataSource(data, group)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *GroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenGroupDataSource(base GroupDataSourceModel, group sdk.Group) (GroupDataSourceModel, diag.Diagnostics) {
	description := types.StringNull()
	if group.Description != nil {
		description = types.StringValue(*group.Description)
	}

	metadata := types.MapNull(types.StringType)
	if group.Metadata != nil {
		stringMetadata := make(map[string]string, len(*group.Metadata))
		for k, v := range *group.Metadata {
			stringMetadata[k] = fmt.Sprintf("%v", v)
		}

		mv, diags := types.MapValueFrom(context.Background(), types.StringType, stringMetadata)
		if diags.HasError() {
			return GroupDataSourceModel{}, diags
		}
		metadata = mv
	}

	createdAt := types.StringNull()
	if !group.CreatedAt.IsZero() {
		createdAt = types.StringValue(group.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !group.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(group.UpdatedAt.Format(time.RFC3339))
	}

	return GroupDataSourceModel{
		OrgName:        base.OrgName,
		Name:           types.StringValue(group.Name),
		ID:             types.StringValue(group.Id.String()),
		Description:    description,
		Metadata:       metadata,
		OrganizationID: types.StringValue(group.OrganizationId.String()),
		OwnerUserID:    types.StringValue(group.OwnerUserId.String()),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}
