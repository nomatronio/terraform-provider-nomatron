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

var _ datasource.DataSource = &JobDataSource{}

type JobDataSource struct {
	client *sdk.ClientWithResponses
}

func NewJobDataSource() datasource.DataSource {
	return &JobDataSource{}
}

type JobDataSourceModel struct {
	OrgName          types.String `tfsdk:"org_name"`
	AppSlug          types.String `tfsdk:"app_slug"`
	Slug             types.String `tfsdk:"slug"`
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	ClusterID        types.String `tfsdk:"cluster_id"`
	DefaultNamespace types.String `tfsdk:"default_namespace"`
	JobspecPath      types.String `tfsdk:"jobspec_path"`
	JobspecType      types.String `tfsdk:"jobspec_type"`
	IsPrimary        types.Bool   `tfsdk:"is_primary"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (d *JobDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (d *JobDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron application job by organization name, application slug, and exact job slug.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the application.",
			},
			"app_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Application slug that owns the job.",
			},
			"slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact job slug.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Job ID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Job display name.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Job description.",
			},
			"cluster_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Optional cluster override for this job.",
			},
			"default_namespace": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Default Nomad namespace for this job.",
			},
			"jobspec_path": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Path to the job spec within the repository.",
			},
			"jobspec_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Job spec source type.",
			},
			"is_primary": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is the primary job for the application.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the job was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the job was last updated.",
			},
		},
	}
}

func (d *JobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data JobDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the job data source.",
		)
		return
	}

	job, err := readAppJob(ctx, d.client, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Job", err.Error())
		return
	}

	data = flattenJobDataSource(data, job)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *JobDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenJobDataSource(base JobDataSourceModel, job sdk.AppJob) JobDataSourceModel {
	description := types.StringNull()
	if job.Description != nil {
		description = types.StringValue(*job.Description)
	}

	clusterID := types.StringNull()
	if job.ClusterID != nil {
		clusterID = types.StringValue(job.ClusterID.String())
	}

	defaultNamespace := types.StringNull()
	if job.DefaultNamespace != "" {
		defaultNamespace = types.StringValue(job.DefaultNamespace)
	}

	createdAt := types.StringNull()
	if !job.CreatedAt.IsZero() {
		createdAt = types.StringValue(job.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !job.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(job.UpdatedAt.Format(time.RFC3339))
	}

	return JobDataSourceModel{
		OrgName:          base.OrgName,
		AppSlug:          base.AppSlug,
		Slug:             base.Slug,
		ID:               types.StringValue(job.ID.String()),
		Name:             types.StringValue(job.Name),
		Description:      description,
		ClusterID:        clusterID,
		DefaultNamespace: defaultNamespace,
		JobspecPath:      types.StringValue(job.JobspecPath),
		JobspecType:      types.StringValue(job.JobspecType),
		IsPrimary:        types.BoolValue(job.IsPrimary),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}
