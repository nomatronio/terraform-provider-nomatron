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
	OrgName       types.String `tfsdk:"org_name"`
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	AppID         types.String `tfsdk:"app_id"`
	AppSlug       types.String `tfsdk:"app_slug"`
	ClientID      types.String `tfsdk:"client_id"`
	PrivateKeyPem types.String `tfsdk:"private_key_pem"`
	WebhookSecret types.String `tfsdk:"webhook_secret"`
	Scope         types.String `tfsdk:"scope"`
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
		AppId:         plan.AppID.ValueString(),
		AppSlug:       plan.AppSlug.ValueString(),
		ClientId:      plan.ClientID.ValueString(),
		Name:          plan.Name.ValueString(),
		Scope:         sdk.CreateGitHubConnectionRequestScopeOrg,
		PrivateKeyPem: stringPointerFromConfig(config.PrivateKeyPem),
		WebhookSecret: stringPointerFromConfig(config.WebhookSecret),
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
		ID:       base.ID,
		Name:     base.Name,
		AppID:    base.AppID,
		AppSlug:  base.AppSlug,
		ClientID: base.ClientID,
		Scope:    base.Scope,
	}, connection)

	return OrganizationGitHubAppIntegrationResourceModel{
		OrgName:  base.OrgName,
		ID:       state.ID,
		Name:     state.Name,
		AppID:    state.AppID,
		AppSlug:  state.AppSlug,
		ClientID: state.ClientID,
		Scope:    state.Scope,
	}
}
