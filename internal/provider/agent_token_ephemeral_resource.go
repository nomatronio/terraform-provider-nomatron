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

var _ ephemeral.EphemeralResource = &AgentTokenEphemeralResource{}
var _ ephemeral.EphemeralResourceWithConfigure = &AgentTokenEphemeralResource{}

type AgentTokenEphemeralResource struct {
	client *sdk.ClientWithResponses
}

func NewAgentTokenEphemeralResource() ephemeral.EphemeralResource {
	return &AgentTokenEphemeralResource{}
}

type AgentTokenEphemeralResourceModel struct {
	AgentID        types.String `tfsdk:"agent_id"`
	Name           types.String `tfsdk:"name"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
	RevokeExisting types.Bool   `tfsdk:"revoke_existing"`
	Token          types.String `tfsdk:"token"`
	TokenID        types.String `tfsdk:"token_id"`
	TokenPrefix    types.String `tfsdk:"token_prefix"`
	RevokedTokens  types.Int64  `tfsdk:"revoked_tokens"`
	RotatedAt      types.String `tfsdk:"rotated_at"`
}

func (r *AgentTokenEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_token"
}

func (r *AgentTokenEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = eschema.Schema{
		MarkdownDescription: "Rotate and return a fresh Nomatron agent token without storing it in Terraform state.",
		Attributes: map[string]eschema.Attribute{
			"agent_id": eschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Agent ID to rotate a token for.",
			},
			"name": eschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional display name for the minted token.",
			},
			"expires_at": eschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional RFC3339 expiration timestamp for the minted token.",
			},
			"revoke_existing": eschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether existing tokens should be revoked as part of rotation.",
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
			"revoked_tokens": eschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of tokens revoked during rotation.",
			},
			"rotated_at": eschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when rotation occurred.",
			},
		},
	}
}

func (r *AgentTokenEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *AgentTokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data AgentTokenEphemeralResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the agent token ephemeral resource.",
		)
		return
	}

	agentID, err := parseAgentID(data.AgentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Agent ID", err.Error())
		return
	}

	body := sdk.RotateNetworkAgentTokenJSONRequestBody{}

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

	if !data.RevokeExisting.IsNull() && !data.RevokeExisting.IsUnknown() {
		revokeExisting := data.RevokeExisting.ValueBool()
		body.RevokeExisting = &revokeExisting
	}

	rsp, err := r.client.RotateNetworkAgentTokenWithResponse(ctx, agentID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Rotate Agent Token", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Rotate Agent Token", formatErrorEnvelope(rsp.JSON400))
		return
	}

	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatErrorEnvelope(rsp.JSON401))
		return
	}

	if rsp.JSON404 != nil {
		resp.Diagnostics.AddError("Agent Not Found", formatErrorEnvelope(rsp.JSON404))
		return
	}

	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Failed To Rotate Agent Token", formatErrorEnvelope(rsp.JSON409))
		return
	}

	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when rotating token for agent %q, got %s.", data.AgentID.ValueString(), rsp.Status()),
		)
		return
	}

	data.Token = types.StringValue(rsp.JSON200.Data.Minted.Token)
	data.TokenID = types.StringValue(rsp.JSON200.Data.Minted.TokenId.String())
	data.TokenPrefix = types.StringValue(rsp.JSON200.Data.Minted.TokenPrefix)
	data.RevokedTokens = types.Int64Value(rsp.JSON200.Data.RevokedTokens)
	data.RotatedAt = types.StringValue(rsp.JSON200.Data.RotatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
