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
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ datasource.DataSource = &OrganizationDataSource{}

type OrganizationDataSource struct {
	client *sdk.ClientWithResponses
}

func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

type OrganizationDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Metadata    types.Map    `tfsdk:"metadata"`
	OwnerUserID types.String `tfsdk:"owner_user_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *OrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron organization by id or exact name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Organization ID. If omitted, the provider can look up the organization by exact name.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Exact organization name. If omitted, the provider reads the organization by id.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Organization description.",
			},
			"metadata": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Metadata attached to the organization.",
			},
			"owner_user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Owner user ID.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the organization was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the organization was last updated.",
			},
		},
	}
}

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrganizationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization data source.",
		)
		return
	}

	var (
		org sdk.Organization
		err error
	)

	if !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != "" {
		orgID, parseErr := parseAgentID(data.ID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid Organization ID", parseErr.Error())
			return
		}

		org, err = findOrganizationByID(ctx, d.client, orgID)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Read Organization", err.Error())
			return
		}
	} else if !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != "" {
		org, err = readOrganizationByName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed To Read Organization", err.Error())
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Lookup Attribute",
			"Set either id or name to read a single organization.",
		)
		return
	}

	data, diags := flattenOrganizationDataSource(org)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *OrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenOrganizationDataSource(org sdk.Organization) (OrganizationDataSourceModel, diag.Diagnostics) {
	description := types.StringNull()
	if org.Description != nil {
		description = types.StringValue(*org.Description)
	}

	metadata := types.MapNull(types.StringType)
	if org.Metadata != nil {
		stringMetadata := make(map[string]string, len(*org.Metadata))
		for k, v := range *org.Metadata {
			stringMetadata[k] = fmt.Sprintf("%v", v)
		}

		mv, diags := types.MapValueFrom(context.Background(), types.StringType, stringMetadata)
		if diags.HasError() {
			return OrganizationDataSourceModel{}, diags
		}
		metadata = mv
	}

	createdAt := types.StringNull()
	if !org.CreatedAt.IsZero() {
		createdAt = types.StringValue(org.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !org.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(org.UpdatedAt.Format(time.RFC3339))
	}

	return OrganizationDataSourceModel{
		ID:          types.StringValue(org.Id.String()),
		Name:        types.StringValue(org.Name),
		Description: description,
		Metadata:    metadata,
		OwnerUserID: types.StringValue(org.OwnerUserId.String()),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func readOrganizationByName(ctx context.Context, client *sdk.ClientWithResponses, name string) (sdk.Organization, error) {
	rsp, err := client.GetOrganizationWithResponse(ctx, name)
	if err != nil {
		return sdk.Organization{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.Organization{}, fmt.Errorf("no organization was found with name %q", name)
	}

	if rsp.JSON200 == nil {
		return sdk.Organization{}, fmt.Errorf("expected 200 response when reading organization %q, got %s", name, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func findOrganizationByID(ctx context.Context, client *sdk.ClientWithResponses, orgID openapi_types.UUID) (sdk.Organization, error) {
	rsp, err := client.ListOrganizationsWithResponse(ctx, nil)
	if err != nil {
		return sdk.Organization{}, err
	}

	if rsp.JSON200 == nil {
		return sdk.Organization{}, fmt.Errorf("expected 200 response when listing organizations for %q, got %s", orgID.String(), rsp.Status())
	}

	var matches []sdk.Organization
	for _, org := range rsp.JSON200.Data.Items {
		if org.Id == orgID {
			matches = append(matches, org)
		}
	}

	switch len(matches) {
	case 0:
		return sdk.Organization{}, fmt.Errorf("no organization was found with id %q", orgID.String())
	case 1:
		return matches[0], nil
	default:
		return sdk.Organization{}, fmt.Errorf("multiple organizations were found with id %q", orgID.String())
	}
}
