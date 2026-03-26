package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	eschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ ephemeral.EphemeralResource = &ServiceAccountTokenEphemeralResource{}
var _ ephemeral.EphemeralResourceWithConfigure = &ServiceAccountTokenEphemeralResource{}

type ServiceAccountTokenEphemeralResource struct {
	client *sdk.ClientWithResponses
}

func NewServiceAccountTokenEphemeralResource() ephemeral.EphemeralResource {
	return &ServiceAccountTokenEphemeralResource{}
}

type ServiceAccountTokenEphemeralResourceModel struct {
	ServiceAccountID types.String `tfsdk:"service_account_id"`
	Name             types.String `tfsdk:"name"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
	Token            types.String `tfsdk:"token"`
	TokenID          types.String `tfsdk:"token_id"`
	TokenPrefix      types.String `tfsdk:"token_prefix"`
}

func (r *ServiceAccountTokenEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_token"
}

func (r *ServiceAccountTokenEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = eschema.Schema{
		MarkdownDescription: "Mint and return a fresh Nomatron service account token without storing it in Terraform state.",
		Attributes: map[string]eschema.Attribute{
			"service_account_id": eschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Service account ID to mint a token for.",
			},
			"name": eschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional display name for the minted token.",
			},
			"expires_at": eschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional RFC3339 expiration timestamp for the minted token.",
			},
			"token": eschema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The newly minted raw token.",
			},
			"token_id": eschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The minted token ID.",
			},
			"token_prefix": eschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Prefix for the minted token.",
			},
		},
	}
}

func (r *ServiceAccountTokenEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ServiceAccountTokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data ServiceAccountTokenEphemeralResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the service account token ephemeral resource.",
		)
		return
	}

	serviceAccountID, err := parseServiceAccountID(data.ServiceAccountID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Service Account ID", err.Error())
		return
	}

	body := sdk.CreateServiceAccountTokenJSONRequestBody{}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		body.Name = &name
	}

	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() && data.ExpiresAt.ValueString() != "" {
		expiresAt, parseErr := time.Parse(time.RFC3339, data.ExpiresAt.ValueString())
		if parseErr != nil {
			resp.Diagnostics.AddError("Invalid Expires At", fmt.Sprintf("expected RFC3339 timestamp, got %q: %v", data.ExpiresAt.ValueString(), parseErr))
			return
		}
		body.ExpiresAt = &expiresAt
	}

	rsp, err := r.client.CreateServiceAccountTokenWithResponse(ctx, serviceAccountID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Service Account Token", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Service Account Token", formatAPIError(rsp.JSON400))
		return
	}

	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatAPIError(rsp.JSON401))
		return
	}

	if rsp.JSON404 != nil {
		resp.Diagnostics.AddError("Service Account Not Found", formatAPIError(rsp.JSON404))
		return
	}

	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Failed To Create Service Account Token", formatAPIError(rsp.JSON409))
		return
	}

	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating token for service account %q, got %s.", data.ServiceAccountID.ValueString(), rsp.Status()),
		)
		return
	}

	data.ServiceAccountID = types.StringValue(rsp.JSON201.Data.ServiceAccountId.String())
	data.Token = types.StringValue(rsp.JSON201.Data.Token)
	data.TokenID = types.StringValue(rsp.JSON201.Data.TokenId.String())
	data.TokenPrefix = types.StringValue(rsp.JSON201.Data.TokenPrefix)

	if rsp.JSON201.Data.ExpiresAt != nil {
		data.ExpiresAt = types.StringValue(rsp.JSON201.Data.ExpiresAt.Format(time.RFC3339))
	} else if data.ExpiresAt.IsUnknown() {
		data.ExpiresAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
