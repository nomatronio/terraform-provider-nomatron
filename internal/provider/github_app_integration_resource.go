package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &GitHubAppIntegrationResource{}
var _ resource.ResourceWithImportState = &GitHubAppIntegrationResource{}

type GitHubAppIntegrationResource struct {
	client *sdk.ClientWithResponses
}

func NewGitHubAppIntegrationResource() resource.Resource {
	return &GitHubAppIntegrationResource{}
}

type GitHubAppIntegrationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	ProviderKind    types.String `tfsdk:"provider_kind"`
	WebBaseURL      types.String `tfsdk:"web_base_url"`
	APIBaseURL      types.String `tfsdk:"api_base_url"`
	UploadBaseURL   types.String `tfsdk:"upload_base_url"`
	AppID           types.String `tfsdk:"app_id"`
	AppSlug         types.String `tfsdk:"app_slug"`
	ClientID        types.String `tfsdk:"client_id"`
	PrivateKeyPemWO types.String `tfsdk:"private_key_pem_wo"`
	WebhookSecretWO types.String `tfsdk:"webhook_secret_wo"`
	TLSCABundlePEM  types.String `tfsdk:"tls_ca_bundle_pem"`
	Scope           types.String `tfsdk:"scope"`
}

func (r *GitHubAppIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_github_app_integration"
}

func (r *GitHubAppIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron global GitHub App integration resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Integration ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Integration name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_kind": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("github_com"),
				MarkdownDescription: "GitHub provider kind. Use `github_com` for GitHub.com or `enterprise_server` for self-hosted GitHub Enterprise Server.",
				Validators: []validator.String{
					stringvalidator.OneOf(string(sdk.GithubCom), string(sdk.EnterpriseServer)),
				},
			},
			"web_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "GitHub web base URL. Required when `provider_kind` is `enterprise_server`; defaults server-side for `github_com`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "GitHub API base URL. Defaults server-side to the GitHub.com API, or to `<web_base_url>/api/v3` for Enterprise when omitted.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"upload_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "GitHub upload API base URL. Defaults server-side to the GitHub.com uploads API, or to `<web_base_url>/api/uploads` for Enterprise when omitted.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "GitHub App ID.",
			},
			"app_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "GitHub App slug.",
			},
			"client_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "GitHub App client ID.",
			},
			"private_key_pem_wo": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "GitHub App private key PEM.",
			},
			"webhook_secret_wo": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "GitHub App webhook secret.",
			},
			"tls_ca_bundle_pem": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Optional PEM-encoded CA bundle for GitHub Enterprise Server instances using a private CA.",
			},
			"scope": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Integration scope, always `global` for this resource.",
			},
		},
	}
}

func (r *GitHubAppIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GitHubAppIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GitHubAppIntegrationResourceModel
	var config GitHubAppIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the github app integration resource.",
		)
		return
	}

	body := sdk.CreateGlobalGitHubIntegrationJSONRequestBody{
		ApiBaseUrl:     stringPointerFromConfig(plan.APIBaseURL),
		AppId:          plan.AppID.ValueString(),
		AppSlug:        plan.AppSlug.ValueString(),
		ClientId:       plan.ClientID.ValueString(),
		Name:           plan.Name.ValueString(),
		Scope:          sdk.CreateGitHubConnectionRequestScopeGlobal,
		PrivateKeyPem:  stringPointerFromConfig(config.PrivateKeyPemWO),
		ProviderKind:   githubProviderKindPointerFromConfig(plan.ProviderKind),
		TlsCaBundlePem: stringPointerFromConfig(plan.TLSCABundlePEM),
		UploadBaseUrl:  stringPointerFromConfig(plan.UploadBaseURL),
		WebBaseUrl:     stringPointerFromConfig(plan.WebBaseURL),
		WebhookSecret:  stringPointerFromConfig(config.WebhookSecretWO),
	}

	rsp, err := r.client.CreateGlobalGitHubIntegrationWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create GitHub App Integration", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create GitHub App Integration", formatAPIError(rsp.JSON400))
		return
	}
	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatAPIError(rsp.JSON401))
		return
	}
	if rsp.JSON403 != nil {
		resp.Diagnostics.AddError("Forbidden", formatAPIError(rsp.JSON403))
		return
	}
	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("GitHub App Integration Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Create GitHub App Integration", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating github app integration, got %s.", rsp.Status()),
		)
		return
	}

	state := stateFromGitHubConnection(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GitHubAppIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GitHubAppIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the github app integration resource.",
		)
		return
	}

	connection, err := readGlobalGitHubIntegration(ctx, r.client, state.Name.ValueString())
	if err != nil {
		if isGitHubIntegrationNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read GitHub App Integration", err.Error())
		return
	}

	newState := stateFromGitHubConnection(state, connection)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *GitHubAppIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GitHubAppIntegrationResourceModel
	var state GitHubAppIntegrationResourceModel
	var config GitHubAppIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the github app integration resource.",
		)
		return
	}

	body := sdk.UpdateGlobalGitHubIntegrationJSONRequestBody{}
	if stringValueChanged(plan.ProviderKind, state.ProviderKind) {
		body.ProviderKind = githubProviderKindPointerFromConfig(plan.ProviderKind)
	}
	if stringValueChanged(plan.WebBaseURL, state.WebBaseURL) {
		body.WebBaseUrl = stringPointerFromConfig(plan.WebBaseURL)
	}
	if stringValueChanged(plan.APIBaseURL, state.APIBaseURL) {
		body.ApiBaseUrl = stringPointerFromConfig(plan.APIBaseURL)
	}
	if stringValueChanged(plan.UploadBaseURL, state.UploadBaseURL) {
		body.UploadBaseUrl = stringPointerFromConfig(plan.UploadBaseURL)
	}
	if stringValueChanged(plan.AppID, state.AppID) {
		appID := plan.AppID.ValueString()
		body.AppId = &appID
	}
	if stringValueChanged(plan.AppSlug, state.AppSlug) {
		appSlug := plan.AppSlug.ValueString()
		body.AppSlug = &appSlug
	}
	if stringValueChanged(plan.ClientID, state.ClientID) {
		clientID := plan.ClientID.ValueString()
		body.ClientId = &clientID
	}
	if !config.PrivateKeyPemWO.IsNull() && !config.PrivateKeyPemWO.IsUnknown() {
		privateKey := config.PrivateKeyPemWO.ValueString()
		body.PrivateKeyPem = &privateKey
	}
	if !config.WebhookSecretWO.IsNull() && !config.WebhookSecretWO.IsUnknown() {
		webhookSecret := config.WebhookSecretWO.ValueString()
		body.WebhookSecret = &webhookSecret
	}
	if stringValueChanged(plan.TLSCABundlePEM, state.TLSCABundlePEM) {
		body.TlsCaBundlePem = stringPointerFromConfig(plan.TLSCABundlePEM)
	}

	rsp, err := r.client.UpdateGlobalGitHubIntegrationWithResponse(ctx, state.Name.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update GitHub App Integration", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update GitHub App Integration", formatAPIError(rsp.JSON400))
		return
	}
	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatAPIError(rsp.JSON401))
		return
	}
	if rsp.JSON403 != nil {
		resp.Diagnostics.AddError("Forbidden", formatAPIError(rsp.JSON403))
		return
	}
	if rsp.JSON404 != nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Update GitHub App Integration", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating github app integration %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}

	newState := stateFromGitHubConnection(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *GitHubAppIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GitHubAppIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the github app integration resource.",
		)
		return
	}

	rsp, err := r.client.DeleteGlobalGitHubIntegrationWithResponse(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete GitHub App Integration", err.Error())
		return
	}

	if rsp.JSON404 != nil {
		return
	}
	if rsp.JSON401 != nil {
		resp.Diagnostics.AddError("Unauthorized", formatAPIError(rsp.JSON401))
		return
	}
	if rsp.JSON403 != nil {
		resp.Diagnostics.AddError("Forbidden", formatAPIError(rsp.JSON403))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Delete GitHub App Integration", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting github app integration %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *GitHubAppIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

type gitHubIntegrationNotFoundError struct {
	name string
}

func (e *gitHubIntegrationNotFoundError) Error() string {
	return fmt.Sprintf("github app integration %q not found", e.name)
}

func isGitHubIntegrationNotFound(err error) bool {
	_, ok := err.(*gitHubIntegrationNotFoundError)
	return ok
}

func readGlobalGitHubIntegration(ctx context.Context, client *sdk.ClientWithResponses, name string) (sdk.GitHubConnection, error) {
	rsp, err := client.GetGlobalGitHubIntegrationWithResponse(ctx, name, nil)
	if err != nil {
		return sdk.GitHubConnection{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.GitHubConnection{}, &gitHubIntegrationNotFoundError{name: name}
	}
	if rsp.JSON200 == nil {
		return sdk.GitHubConnection{}, fmt.Errorf("expected 200 response when reading github app integration %q, got %s", name, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func stateFromGitHubConnection(base GitHubAppIntegrationResourceModel, connection sdk.GitHubConnection) GitHubAppIntegrationResourceModel {
	appID := base.AppID
	if appID.IsUnknown() || appID.IsNull() {
		appID = types.StringNull()
	}
	if connection.AppId != nil {
		appID = types.StringValue(*connection.AppId)
	}

	appSlug := base.AppSlug
	if appSlug.IsUnknown() || appSlug.IsNull() {
		appSlug = types.StringNull()
	}
	if connection.AppSlug != nil {
		appSlug = types.StringValue(*connection.AppSlug)
	}

	clientID := base.ClientID
	if clientID.IsUnknown() || clientID.IsNull() {
		clientID = types.StringNull()
	}
	if connection.ClientId != nil {
		clientID = types.StringValue(*connection.ClientId)
	}
	providerKind := base.ProviderKind
	if connection.ProviderKind != nil {
		providerKind = types.StringValue(string(*connection.ProviderKind))
	} else if providerKind.IsUnknown() || providerKind.IsNull() {
		providerKind = types.StringNull()
	}
	webBaseURL := stringFromOptional(base.WebBaseURL, connection.WebBaseUrl)
	apiBaseURL := stringFromOptional(base.APIBaseURL, connection.ApiBaseUrl)
	uploadBaseURL := stringFromOptional(base.UploadBaseURL, connection.UploadBaseUrl)

	return GitHubAppIntegrationResourceModel{
		ID:             types.StringValue(connection.Id.String()),
		Name:           types.StringValue(connection.Name),
		ProviderKind:   providerKind,
		WebBaseURL:     webBaseURL,
		APIBaseURL:     apiBaseURL,
		UploadBaseURL:  uploadBaseURL,
		AppID:          appID,
		AppSlug:        appSlug,
		ClientID:       clientID,
		TLSCABundlePEM: base.TLSCABundlePEM,
		Scope:          types.StringValue(string(connection.Scope)),
	}
}

func stringPointerFromConfig(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func githubProviderKindPointerFromConfig(v types.String) *sdk.GitHubProviderKind {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	kind := sdk.GitHubProviderKind(v.ValueString())
	return &kind
}

func stringFromOptional(base types.String, value *string) types.String {
	if value != nil {
		return types.StringValue(*value)
	}
	if base.IsUnknown() || base.IsNull() {
		return types.StringNull()
	}
	return base
}
