package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	AppID         types.String `tfsdk:"app_id"`
	AppSlug       types.String `tfsdk:"app_slug"`
	ClientID      types.String `tfsdk:"client_id"`
	PrivateKeyPem types.String `tfsdk:"private_key_pem"`
	WebhookSecret types.String `tfsdk:"webhook_secret"`
	Scope         types.String `tfsdk:"scope"`
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
			"private_key_pem": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "GitHub App private key PEM.",
			},
			"webhook_secret": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "GitHub App webhook secret.",
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
		AppId:         plan.AppID.ValueString(),
		AppSlug:       plan.AppSlug.ValueString(),
		ClientId:      plan.ClientID.ValueString(),
		Name:          plan.Name.ValueString(),
		Scope:         sdk.CreateGitHubConnectionRequestScopeGlobal,
		PrivateKeyPem: stringPointerFromConfig(config.PrivateKeyPem),
		WebhookSecret: stringPointerFromConfig(config.WebhookSecret),
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
	if !config.PrivateKeyPem.IsNull() && !config.PrivateKeyPem.IsUnknown() {
		privateKey := config.PrivateKeyPem.ValueString()
		body.PrivateKeyPem = &privateKey
	}
	if !config.WebhookSecret.IsNull() && !config.WebhookSecret.IsUnknown() {
		webhookSecret := config.WebhookSecret.ValueString()
		body.WebhookSecret = &webhookSecret
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

	return GitHubAppIntegrationResourceModel{
		ID:       types.StringValue(connection.Id.String()),
		Name:     types.StringValue(connection.Name),
		AppID:    appID,
		AppSlug:  appSlug,
		ClientID: clientID,
		Scope:    types.StringValue(string(connection.Scope)),
	}
}

func stringPointerFromConfig(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}
