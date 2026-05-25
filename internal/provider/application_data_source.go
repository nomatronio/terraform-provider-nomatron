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

var _ datasource.DataSource = &ApplicationDataSource{}

type ApplicationDataSource struct {
	client *sdk.ClientWithResponses
}

func NewApplicationDataSource() datasource.DataSource {
	return &ApplicationDataSource{}
}

type ApplicationDataSourceModel struct {
	OrgName        types.String `tfsdk:"org_name"`
	Slug           types.String `tfsdk:"slug"`
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	ClusterID      types.String `tfsdk:"cluster_id"`
	RepoURL        types.String `tfsdk:"repo_url"`
	GitProvider    types.String `tfsdk:"git_provider"`
	Ref            types.String `tfsdk:"ref"`
	TriggerMode    types.String `tfsdk:"trigger_mode"`
	TagPatternType types.String `tfsdk:"tag_pattern_type"`
	TagPattern     types.String `tfsdk:"tag_pattern"`
	VcsGitHubID    types.String `tfsdk:"vcs_github_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	UpdatedBy      types.String `tfsdk:"updated_by"`
}

func (d *ApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *ApplicationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron application by organization name and exact slug.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the application.",
			},
			"slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact application slug.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application ID.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application display name.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application description.",
			},
			"cluster_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Nomad cluster ID backing the application.",
			},
			"repo_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source repository URL.",
			},
			"git_provider": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Git provider for the repository.",
			},
			"ref": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Branch ref tracked when the application uses branch commit triggers.",
			},
			"trigger_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Deployment trigger mode: `branch_commit` or `tag`.",
			},
			"tag_pattern_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Tag matching style for tag-based triggers.",
			},
			"tag_pattern": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Prefix, suffix, or regular expression used by tag-based triggers.",
			},
			"vcs_github_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "GitHub App integration ID when connected.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the application was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the application was last updated.",
			},
			"updated_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Actor ID that last updated the application, when available.",
			},
		},
	}
}

func (d *ApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the application data source.",
		)
		return
	}

	app, err := readApplication(ctx, d.client, data.OrgName.ValueString(), data.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Application", err.Error())
		return
	}

	data = flattenApplicationDataSource(data, app)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ApplicationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenApplicationDataSource(base ApplicationDataSourceModel, app sdk.App) ApplicationDataSourceModel {
	description := types.StringNull()
	if app.Description != nil {
		description = types.StringValue(*app.Description)
	}

	ref := types.StringNull()
	if app.Ref != nil {
		ref = types.StringValue(*app.Ref)
	}
	triggerMode := types.StringNull()
	if app.TriggerMode != nil {
		triggerMode = types.StringValue(string(*app.TriggerMode))
	}
	tagPatternType := types.StringNull()
	if app.TagPatternType != nil {
		tagPatternType = types.StringValue(string(*app.TagPatternType))
	}
	tagPattern := types.StringNull()
	if app.TagPattern != nil {
		tagPattern = types.StringValue(*app.TagPattern)
	}

	vcsGitHubID := types.StringNull()
	if app.VcsGithubId != nil {
		vcsGitHubID = types.StringValue(app.VcsGithubId.String())
	}

	createdAt := types.StringNull()
	if !app.CreatedAt.IsZero() {
		createdAt = types.StringValue(app.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !app.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(app.UpdatedAt.Format(time.RFC3339))
	}

	updatedBy := types.StringNull()
	if app.UpdatedBy != nil {
		updatedBy = types.StringValue(app.UpdatedBy.String())
	}

	return ApplicationDataSourceModel{
		OrgName:        base.OrgName,
		Slug:           base.Slug,
		ID:             types.StringValue(app.Id.String()),
		Name:           types.StringValue(app.Name),
		Description:    description,
		ClusterID:      types.StringValue(app.ClusterId.String()),
		RepoURL:        types.StringValue(app.RepoUrl),
		GitProvider:    types.StringValue(string(app.GitProvider)),
		Ref:            ref,
		TriggerMode:    triggerMode,
		TagPatternType: tagPatternType,
		TagPattern:     tagPattern,
		VcsGitHubID:    vcsGitHubID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		UpdatedBy:      updatedBy,
	}
}
