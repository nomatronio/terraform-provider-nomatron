package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &OIDCProviderResource{}
var _ resource.ResourceWithImportState = &OIDCProviderResource{}

type OIDCProviderResource struct {
	client *sdk.ClientWithResponses
}

func NewOIDCProviderResource() resource.Resource {
	return &OIDCProviderResource{}
}

type OIDCProviderResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Slug                types.String `tfsdk:"slug"`
	DisplayName         types.String `tfsdk:"display_name"`
	IssuerURL           types.String `tfsdk:"issuer_url"`
	ClientID            types.String `tfsdk:"client_id"`
	ClientSecretWO      types.String `tfsdk:"client_secret_wo"`
	Scopes              types.List   `tfsdk:"scopes"`
	UsernameClaim       types.String `tfsdk:"username_claim"`
	EmailClaim          types.String `tfsdk:"email_claim"`
	NameClaim           types.String `tfsdk:"name_claim"`
	GroupsClaim         types.String `tfsdk:"groups_claim"`
	AllowedEmailDomains types.List   `tfsdk:"allowed_email_domains"`
	AutoProvision       types.Bool   `tfsdk:"auto_provision"`
	SyncProfile         types.Bool   `tfsdk:"sync_profile"`
	SyncGroups          types.Bool   `tfsdk:"sync_groups"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	IsDefault           types.Bool   `tfsdk:"is_default"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func (r *OIDCProviderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_provider"
}

func (r *OIDCProviderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron OIDC provider resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OIDC provider ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Stable OIDC provider slug.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable provider name.",
			},
			"issuer_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OIDC issuer URL.",
			},
			"client_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OIDC client ID.",
			},
			"client_secret_wo": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "OIDC client secret. This value is write-only and is never stored in Terraform state.",
			},
			"scopes": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "OIDC scopes requested during login.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"username_claim": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Claim used for the Nomatron username.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email_claim": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Claim used for the user's email address.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name_claim": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Claim used for the display name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"groups_claim": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Claim containing external group values.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allowed_email_domains": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional allow-list of email domains permitted to authenticate through this provider.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_provision": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether valid first-time OIDC logins auto-create Nomatron users.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"sync_profile": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether profile fields are synchronized at login.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"sync_groups": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether external group claims synchronize mapped RBAC assignments at login.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this provider can be used for login.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_default": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this provider is the default OIDC login provider.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the provider was last updated.",
			},
		},
	}
}

func (r *OIDCProviderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *OIDCProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OIDCProviderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC provider resource.")
		return
	}

	body := sdk.CreateOIDCProviderJSONRequestBody{
		Slug:        plan.Slug.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
		IssuerUrl:   plan.IssuerURL.ValueString(),
		ClientId:    plan.ClientID.ValueString(),
	}

	if !plan.ClientSecretWO.IsNull() && !plan.ClientSecretWO.IsUnknown() && plan.ClientSecretWO.ValueString() != "" {
		secret := plan.ClientSecretWO.ValueString()
		body.ClientSecret = &secret
	}

	resp.Diagnostics.Append(populateCreateOIDCProviderBody(ctx, &body, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.CreateOIDCProviderWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create OIDC Provider", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create OIDC Provider", formatAPIError(rsp.JSON400))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError("Unexpected API Response", fmt.Sprintf("Expected 201 response when creating OIDC provider %q, got %s.", plan.Slug.ValueString(), rsp.Status()))
		return
	}

	state, diags := stateFromOIDCProvider(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OIDCProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OIDCProviderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC provider resource.")
		return
	}

	provider, err := readOIDCProvider(ctx, r.client, state.Slug.ValueString())
	if err != nil {
		if isOIDCProviderNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read OIDC Provider", err.Error())
		return
	}

	newState, diags := stateFromOIDCProvider(state, provider)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OIDCProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OIDCProviderResourceModel
	var state OIDCProviderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC provider resource.")
		return
	}

	body := sdk.UpdateOIDCProviderJSONRequestBody{}
	populateUpdateOIDCProviderBody(ctx, &body, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.UpdateOIDCProviderWithResponse(ctx, state.Slug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update OIDC Provider", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError("Unexpected API Response", fmt.Sprintf("Expected 200 response when updating OIDC provider %q, got %s.", state.Slug.ValueString(), rsp.Status()))
		return
	}

	if !plan.ClientSecretWO.IsNull() && !plan.ClientSecretWO.IsUnknown() && plan.ClientSecretWO.ValueString() != "" {
		secret := plan.ClientSecretWO.ValueString()
		secretResp, err := r.client.RotateOIDCProviderSecretWithResponse(ctx, state.Slug.ValueString(), sdk.RotateOIDCProviderSecretJSONRequestBody{
			ClientSecret: &secret,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed To Rotate OIDC Provider Secret", err.Error())
			return
		}
		if secretResp.JSON400 != nil {
			resp.Diagnostics.AddError("Failed To Rotate OIDC Provider Secret", formatAPIError(secretResp.JSON400))
			return
		}
		if secretResp.JSON200 == nil {
			resp.Diagnostics.AddError("Unexpected API Response", fmt.Sprintf("Expected 200 response when rotating OIDC provider secret for %q, got %s.", state.Slug.ValueString(), secretResp.Status()))
			return
		}
	}

	provider, err := readOIDCProvider(ctx, r.client, state.Slug.ValueString())
	if err != nil {
		if isOIDCProviderNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read OIDC Provider", err.Error())
		return
	}

	newState, diags := stateFromOIDCProvider(plan, provider)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OIDCProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OIDCProviderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC provider resource.")
		return
	}

	rsp, err := r.client.DeleteOIDCProviderWithResponse(ctx, state.Slug.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete OIDC Provider", err.Error())
		return
	}
	if rsp.JSON404 != nil {
		return
	}
	if rsp.StatusCode() != http.StatusOK && rsp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError("Unexpected API Response", fmt.Sprintf("Expected 200/204 response when deleting OIDC provider %q, got %s.", state.Slug.ValueString(), rsp.Status()))
		return
	}
}

func (r *OIDCProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("slug"), req, resp)
}

func populateCreateOIDCProviderBody(ctx context.Context, body *sdk.CreateOIDCProviderJSONRequestBody, plan OIDCProviderResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	body.Scopes, diags = terraformOptionalStringList(ctx, plan.Scopes)
	if diags.HasError() {
		return diags
	}
	body.AllowedEmailDomains, diags = terraformOptionalStringList(ctx, plan.AllowedEmailDomains)
	if diags.HasError() {
		return diags
	}

	body.UsernameClaim = terraformOptionalString(plan.UsernameClaim)
	body.EmailClaim = terraformOptionalString(plan.EmailClaim)
	body.NameClaim = terraformOptionalString(plan.NameClaim)
	body.GroupsClaim = terraformOptionalString(plan.GroupsClaim)
	body.AutoProvision = terraformOptionalBool(plan.AutoProvision)
	body.SyncProfile = terraformOptionalBool(plan.SyncProfile)
	body.SyncGroups = terraformOptionalBool(plan.SyncGroups)
	body.Enabled = terraformOptionalBool(plan.Enabled)
	body.IsDefault = terraformOptionalBool(plan.IsDefault)

	return diags
}

func populateUpdateOIDCProviderBody(ctx context.Context, body *sdk.UpdateOIDCProviderJSONRequestBody, plan OIDCProviderResourceModel, diags *diag.Diagnostics) {
	body.DisplayName = terraformOptionalString(plan.DisplayName)
	body.IssuerUrl = terraformOptionalString(plan.IssuerURL)
	body.ClientId = terraformOptionalString(plan.ClientID)
	body.UsernameClaim = terraformOptionalString(plan.UsernameClaim)
	body.EmailClaim = terraformOptionalString(plan.EmailClaim)
	body.NameClaim = terraformOptionalString(plan.NameClaim)
	body.GroupsClaim = terraformOptionalString(plan.GroupsClaim)
	body.AutoProvision = terraformOptionalBool(plan.AutoProvision)
	body.SyncProfile = terraformOptionalBool(plan.SyncProfile)
	body.SyncGroups = terraformOptionalBool(plan.SyncGroups)
	body.Enabled = terraformOptionalBool(plan.Enabled)
	body.IsDefault = terraformOptionalBool(plan.IsDefault)

	scopes, listDiags := terraformOptionalStringList(ctx, plan.Scopes)
	diags.Append(listDiags...)
	body.Scopes = scopes

	allowedDomains, listDiags := terraformOptionalStringList(ctx, plan.AllowedEmailDomains)
	diags.Append(listDiags...)
	body.AllowedEmailDomains = allowedDomains
}

func stateFromOIDCProvider(base OIDCProviderResourceModel, provider sdk.OIDCProvider) (OIDCProviderResourceModel, diag.Diagnostics) {
	scopes, diags := stringListState(provider.Scopes)
	if diags.HasError() {
		return OIDCProviderResourceModel{}, diags
	}

	allowedDomains, diags := stringListState(provider.AllowedEmailDomains)
	if diags.HasError() {
		return OIDCProviderResourceModel{}, diags
	}

	updatedAt := types.StringNull()
	if provider.UpdatedAt != nil && !provider.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(provider.UpdatedAt.Format(time.RFC3339))
	}

	return OIDCProviderResourceModel{
		ID:                  types.StringValue(provider.Id.String()),
		Slug:                types.StringValue(provider.Slug),
		DisplayName:         types.StringValue(provider.DisplayName),
		IssuerURL:           stringState(provider.IssuerUrl),
		ClientID:            stringState(provider.ClientId),
		ClientSecretWO:      types.StringNull(),
		Scopes:              scopes,
		UsernameClaim:       stringState(provider.UsernameClaim),
		EmailClaim:          stringState(provider.EmailClaim),
		NameClaim:           stringState(provider.NameClaim),
		GroupsClaim:         stringState(provider.GroupsClaim),
		AllowedEmailDomains: allowedDomains,
		AutoProvision:       boolState(provider.AutoProvision),
		SyncProfile:         boolState(provider.SyncProfile),
		SyncGroups:          boolState(provider.SyncGroups),
		Enabled:             types.BoolValue(provider.Enabled),
		IsDefault:           types.BoolValue(provider.IsDefault),
		UpdatedAt:           updatedAt,
	}, nil
}

type oidcProviderNotFoundError struct {
	slug string
}

func (e *oidcProviderNotFoundError) Error() string {
	return fmt.Sprintf("OIDC provider %q not found", e.slug)
}

func isOIDCProviderNotFound(err error) bool {
	_, ok := err.(*oidcProviderNotFoundError)
	return ok
}

func readOIDCProvider(ctx context.Context, client *sdk.ClientWithResponses, slug string) (sdk.OIDCProvider, error) {
	rsp, err := client.GetOIDCProviderWithResponse(ctx, slug)
	if err != nil {
		return sdk.OIDCProvider{}, err
	}
	if rsp.JSON404 != nil {
		return sdk.OIDCProvider{}, &oidcProviderNotFoundError{slug: slug}
	}
	if rsp.JSON200 == nil {
		return sdk.OIDCProvider{}, fmt.Errorf("expected 200 response when reading OIDC provider %q, got %s", slug, rsp.Status())
	}
	return rsp.JSON200.Data, nil
}

func terraformOptionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	out := value.ValueString()
	return &out
}

func terraformOptionalBool(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	out := value.ValueBool()
	return &out
}

func terraformOptionalStringList(ctx context.Context, value types.List) (*[]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	out, diags := terraformListToStringSlice(ctx, value)
	if diags.HasError() {
		return nil, diags
	}
	return &out, nil
}

func stringState(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func boolState(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func stringListState(values *[]string) (types.List, diag.Diagnostics) {
	if values == nil {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(context.Background(), types.StringType, *values)
}
