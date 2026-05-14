package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &OrganizationGitHubAppIntegrationResource{}
var _ resource.ResourceWithImportState = &OrganizationGitHubAppIntegrationResource{}

type OrganizationGitHubAppIntegrationResource struct {
	client *sdk.ClientWithResponses
}

func NewOrganizationGitHubAppIntegrationResource() resource.Resource {
	return &OrganizationGitHubAppIntegrationResource{}
}

type OrganizationGitHubAppIntegrationResourceModel struct {
	OrgName         types.String `tfsdk:"org_name"`
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

func (r *OrganizationGitHubAppIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_github_app_integration"
}

func (r *OrganizationGitHubAppIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron organization-scoped GitHub App integration resource.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the integration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
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
			},
			"web_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("https://github.com"),
				MarkdownDescription: "GitHub web base URL. Required for GitHub Enterprise Server.",
			},
			"api_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("https://api.github.com"),
				MarkdownDescription: "GitHub API base URL. Defaults to the GitHub.com API, or to `/api/v3` for Enterprise when omitted from the API request.",
			},
			"upload_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("https://uploads.github.com"),
				MarkdownDescription: "GitHub upload API base URL. Defaults to the GitHub.com uploads API, or to `/api/uploads` for Enterprise when omitted from the API request.",
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
				MarkdownDescription: "Integration scope, always `org` for this resource.",
			},
		},
	}
}

func (r *OrganizationGitHubAppIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationGitHubAppIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationGitHubAppIntegrationResourceModel
	var config OrganizationGitHubAppIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization github app integration resource.",
		)
		return
	}

	body := sdk.CreateOrganizationGitHubIntegrationJSONRequestBody{
		ApiBaseUrl:     stringPointerFromConfig(plan.APIBaseURL),
		AppId:          plan.AppID.ValueString(),
		AppSlug:        plan.AppSlug.ValueString(),
		ClientId:       plan.ClientID.ValueString(),
		Name:           plan.Name.ValueString(),
		Scope:          sdk.CreateGitHubConnectionRequestScopeOrg,
		PrivateKeyPem:  stringPointerFromConfig(config.PrivateKeyPemWO),
		ProviderKind:   githubProviderKindPointerFromConfig(plan.ProviderKind),
		TlsCaBundlePem: stringPointerFromConfig(plan.TLSCABundlePEM),
		UploadBaseUrl:  stringPointerFromConfig(plan.UploadBaseURL),
		WebBaseUrl:     stringPointerFromConfig(plan.WebBaseURL),
		WebhookSecret:  stringPointerFromConfig(config.WebhookSecretWO),
	}

	rsp, err := r.client.CreateOrganizationGitHubIntegrationWithResponse(ctx, plan.OrgName.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Organization GitHub App Integration", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Organization GitHub App Integration", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Organization Not Found", formatAPIError(rsp.JSON404))
		return
	}
	if rsp.JSON409 != nil {
		resp.Diagnostics.AddError("Organization GitHub App Integration Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Create Organization GitHub App Integration", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating organization github app integration, got %s.", rsp.Status()),
		)
		return
	}

	state := stateFromOrganizationGitHubConnection(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationGitHubAppIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationGitHubAppIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization github app integration resource.",
		)
		return
	}

	connection, err := readOrganizationGitHubIntegration(ctx, r.client, state.OrgName.ValueString(), state.Name.ValueString())
	if err != nil {
		if isOrganizationGitHubIntegrationNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Organization GitHub App Integration", err.Error())
		return
	}

	newState := stateFromOrganizationGitHubConnection(state, connection)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OrganizationGitHubAppIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationGitHubAppIntegrationResourceModel
	var state OrganizationGitHubAppIntegrationResourceModel
	var config OrganizationGitHubAppIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization github app integration resource.",
		)
		return
	}

	body := sdk.UpdateOrganizationGitHubIntegrationJSONRequestBody{}
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

	rsp, err := r.client.UpdateOrganizationGitHubIntegrationWithResponse(ctx, state.OrgName.ValueString(), state.Name.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Organization GitHub App Integration", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Organization GitHub App Integration", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Update Organization GitHub App Integration", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating organization github app integration %q in org %q, got %s.", state.Name.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	newState := stateFromOrganizationGitHubConnection(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OrganizationGitHubAppIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationGitHubAppIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization github app integration resource.",
		)
		return
	}

	rsp, err := r.client.DeleteOrganizationGitHubIntegrationWithResponse(ctx, state.OrgName.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Organization GitHub App Integration", err.Error())
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
		resp.Diagnostics.AddError("Failed To Delete Organization GitHub App Integration", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting organization github app integration %q in org %q, got %s.", state.Name.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *OrganizationGitHubAppIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import identifier in the format `org_name/integration_name`.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

type organizationGitHubIntegrationNotFoundError struct {
	orgName string
	name    string
}

func (e *organizationGitHubIntegrationNotFoundError) Error() string {
	return fmt.Sprintf("organization github app integration %q in org %q not found", e.name, e.orgName)
}

func isOrganizationGitHubIntegrationNotFound(err error) bool {
	_, ok := err.(*organizationGitHubIntegrationNotFoundError)
	return ok
}

func readOrganizationGitHubIntegration(ctx context.Context, client *sdk.ClientWithResponses, orgName, name string) (sdk.GitHubConnection, error) {
	rsp, err := client.GetOrganizationGitHubIntegrationWithResponse(ctx, orgName, name, nil)
	if err != nil {
		return sdk.GitHubConnection{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.GitHubConnection{}, &organizationGitHubIntegrationNotFoundError{orgName: orgName, name: name}
	}
	if rsp.JSON200 == nil {
		return sdk.GitHubConnection{}, fmt.Errorf("expected 200 response when reading organization github app integration %q in org %q, got %s", name, orgName, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func stateFromOrganizationGitHubConnection(base OrganizationGitHubAppIntegrationResourceModel, connection sdk.GitHubConnection) OrganizationGitHubAppIntegrationResourceModel {
	state := stateFromGitHubConnection(GitHubAppIntegrationResourceModel{
		ID:             base.ID,
		Name:           base.Name,
		ProviderKind:   base.ProviderKind,
		WebBaseURL:     base.WebBaseURL,
		APIBaseURL:     base.APIBaseURL,
		UploadBaseURL:  base.UploadBaseURL,
		AppID:          base.AppID,
		AppSlug:        base.AppSlug,
		ClientID:       base.ClientID,
		TLSCABundlePEM: base.TLSCABundlePEM,
		Scope:          base.Scope,
	}, connection)

	return OrganizationGitHubAppIntegrationResourceModel{
		OrgName:        base.OrgName,
		ID:             state.ID,
		Name:           state.Name,
		ProviderKind:   state.ProviderKind,
		WebBaseURL:     state.WebBaseURL,
		APIBaseURL:     state.APIBaseURL,
		UploadBaseURL:  state.UploadBaseURL,
		AppID:          state.AppID,
		AppSlug:        state.AppSlug,
		ClientID:       state.ClientID,
		TLSCABundlePEM: state.TLSCABundlePEM,
		Scope:          state.Scope,
	}
}
