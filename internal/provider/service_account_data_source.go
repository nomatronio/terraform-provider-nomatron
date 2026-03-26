package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ datasource.DataSource = &ServiceAccountDataSource{}

type ServiceAccountDataSource struct {
	client *sdk.ClientWithResponses
}

func NewServiceAccountDataSource() datasource.DataSource {
	return &ServiceAccountDataSource{}
}

type ServiceAccountDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	CreatedAt   types.String `tfsdk:"created_at"`
	CreatedBy   types.String `tfsdk:"created_by"`
}

func (d *ServiceAccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (d *ServiceAccountDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single Nomatron service account by id or exact name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Service account ID. If omitted, the provider can look up the service account by exact name.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Exact service account name. If omitted, the provider reads the service account by id.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Service account description.",
			},
			"is_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the service account is active.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the service account was created.",
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Actor ID that created the service account.",
			},
		},
	}
}

func (d *ServiceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceAccountDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the service account data source.",
		)
		return
	}

	var (
		account sdk.ServiceAccount
		err     error
	)

	if !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != "" {
		accountID, parseErr := parseServiceAccountID(data.ID.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid Service Account ID", parseErr.Error())
			return
		}

		account, err = readServiceAccountByID(ctx, d.client, accountID)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Read Service Account", err.Error())
			return
		}
	} else if !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != "" {
		account, err = findServiceAccountByExactName(ctx, d.client, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed To Read Service Account", err.Error())
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Lookup Attribute",
			"Set either id or name to read a single service account.",
		)
		return
	}

	data = flattenServiceAccountDataSource(account)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ServiceAccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func flattenServiceAccountDataSource(account sdk.ServiceAccount) ServiceAccountDataSourceModel {
	description := types.StringNull()
	if account.Description != nil {
		description = types.StringValue(*account.Description)
	}

	createdAt := types.StringNull()
	if !account.CreatedAt.IsZero() {
		createdAt = types.StringValue(account.CreatedAt.Format(time.RFC3339))
	}

	createdBy := types.StringNull()
	if account.CreatedBy != (openapi_types.UUID{}) {
		createdBy = types.StringValue(account.CreatedBy.String())
	}

	return ServiceAccountDataSourceModel{
		ID:          types.StringValue(account.Id.String()),
		Name:        types.StringValue(account.Name),
		Description: description,
		IsActive:    types.BoolValue(account.IsActive),
		CreatedAt:   createdAt,
		CreatedBy:   createdBy,
	}
}

func readServiceAccountByID(ctx context.Context, client *sdk.ClientWithResponses, accountID openapi_types.UUID) (sdk.ServiceAccount, error) {
	rsp, err := client.ReadServiceAccountWithResponse(ctx, accountID)
	if err != nil {
		return sdk.ServiceAccount{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.ServiceAccount{}, fmt.Errorf("no service account was found with id %q", accountID.String())
	}

	if rsp.JSON200 == nil {
		return sdk.ServiceAccount{}, fmt.Errorf("expected 200 response when reading service account %q, got %s", accountID.String(), rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func findServiceAccountByExactName(ctx context.Context, client *sdk.ClientWithResponses, name string) (sdk.ServiceAccount, error) {
	params := &sdk.ListServiceAccountsParams{
		Name: &name,
	}

	rsp, err := client.ListServiceAccountsWithResponse(ctx, params)
	if err != nil {
		return sdk.ServiceAccount{}, err
	}

	if rsp.JSON200 == nil {
		return sdk.ServiceAccount{}, fmt.Errorf("expected 200 response when listing service accounts for %q, got %s", name, rsp.Status())
	}

	var matches []sdk.ServiceAccount
	for _, account := range rsp.JSON200.Data.Items {
		if account.Name == name {
			matches = append(matches, account)
		}
	}

	switch len(matches) {
	case 0:
		return sdk.ServiceAccount{}, fmt.Errorf("no service account was found with name %q", name)
	case 1:
		return matches[0], nil
	default:
		return sdk.ServiceAccount{}, fmt.Errorf("multiple service accounts were found with name %q", name)
	}
}
