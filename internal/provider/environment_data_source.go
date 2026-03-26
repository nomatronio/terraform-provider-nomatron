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

var _ datasource.DataSource = &EnvironmentDataSource{}

type EnvironmentDataSource struct {
	client *sdk.ClientWithResponses
}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

type EnvironmentDataSourceModel struct {
	OrgName   types.String `tfsdk:"org_name"`
	AppSlug   types.String `tfsdk:"app_slug"`
	JobSlug   types.String `tfsdk:"job_slug"`
	Slug      types.String `tfsdk:"slug"`
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Namespace types.String `tfsdk:"namespace"`
	Priority  types.Int64  `tfsdk:"priority"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron application environment by organization name, application slug, job slug, and exact environment slug.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the application.",
			},
			"app_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application slug that owns the environment.",
			},
			"job_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Job slug that owns the environment.",
			},
			"slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact environment slug.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Environment ID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Environment display name.",
			},
			"cluster_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Nomad cluster ID used by the environment.",
			},
			"namespace": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Nomad namespace for the environment.",
			},
			"priority": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Relative environment priority used by Nomatron.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the environment was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the environment was last updated.",
			},
		},
	}
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the environment data source.",
		)
		return
	}

	env, err := readEnvironment(ctx, d.client, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Environment", err.Error())
		return
	}

	data = flattenEnvironmentDataSource(data, env)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenEnvironmentDataSource(base EnvironmentDataSourceModel, env sdk.Environment) EnvironmentDataSourceModel {
	createdAt := types.StringNull()
	if !env.CreatedAt.IsZero() {
		createdAt = types.StringValue(env.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !env.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(env.UpdatedAt.Format(time.RFC3339))
	}

	return EnvironmentDataSourceModel{
		OrgName:   base.OrgName,
		AppSlug:   base.AppSlug,
		JobSlug:   base.JobSlug,
		Slug:      base.Slug,
		ID:        types.StringValue(env.Id.String()),
		Name:      types.StringValue(env.Name),
		ClusterID: types.StringValue(env.ClusterId.String()),
		Namespace: types.StringValue(env.Namespace),
		Priority:  types.Int64Value(int64(env.Priority)),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
