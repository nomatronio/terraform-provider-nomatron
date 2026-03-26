package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &NomadClusterResource{}
var _ resource.ResourceWithImportState = &NomadClusterResource{}

type NomadClusterResource struct {
	client *sdk.ClientWithResponses
}

func NewNomadClusterResource() resource.Resource {
	return &NomadClusterResource{}
}

type NomadClusterResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	ConnectivityMode types.String `tfsdk:"connectivity_mode"`
	Address          types.String `tfsdk:"address"`
	AgentID          types.String `tfsdk:"agent_id"`
	SkipVerify       types.Bool   `tfsdk:"skip_verify"`
	AclToken         types.String `tfsdk:"acl_token"`
	CaCert           types.String `tfsdk:"ca_cert"`
	TlsCert          types.String `tfsdk:"tls_cert"`
	TlsKey           types.String `tfsdk:"tls_key"`
	Scope            types.String `tfsdk:"scope"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *NomadClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nomad_cluster"
}

func (r *NomadClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron global Nomad cluster resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Cluster name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Cluster description.",
			},
			"connectivity_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Connectivity mode, either `direct` or `agent`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"address": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Nomad API address for `direct` connectivity mode.",
			},
			"agent_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Agent ID for `agent` connectivity mode.",
			},
			"skip_verify": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether TLS verification is skipped for direct connections.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"acl_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "Nomad ACL token used in direct mode.",
			},
			"ca_cert": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "PEM encoded CA certificate for direct mode.",
			},
			"tls_cert": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "PEM encoded client certificate for direct mode.",
			},
			"tls_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "PEM encoded client key for direct mode.",
			},
			"scope": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster scope, typically `global` for this resource.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the cluster was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the cluster was last updated.",
			},
		},
	}
}

func (r *NomadClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NomadClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NomadClusterResourceModel
	var config NomadClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the nomad cluster resource.",
		)
		return
	}

	body, diags := buildCreateClusterBody(plan, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.CreateGlobalClusterWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Nomad Cluster", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Nomad Cluster", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Nomad Cluster Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Create Nomad Cluster", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating nomad cluster, got %s.", rsp.Status()),
		)
		return
	}

	state := stateFromNomadCluster(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *NomadClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NomadClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the nomad cluster resource.",
		)
		return
	}

	cluster, err := readGlobalCluster(ctx, r.client, state.Name.ValueString())
	if err != nil {
		if isNomadClusterNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Nomad Cluster", err.Error())
		return
	}

	newState := stateFromNomadCluster(state, cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *NomadClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NomadClusterResourceModel
	var state NomadClusterResourceModel
	var config NomadClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the nomad cluster resource.",
		)
		return
	}

	body, diags := buildUpdateClusterBody(plan, state, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.UpdateGlobalClusterWithResponse(ctx, state.Name.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Nomad Cluster", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Nomad Cluster", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Update Nomad Cluster", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating nomad cluster %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}

	newState := stateFromNomadCluster(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *NomadClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NomadClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the nomad cluster resource.",
		)
		return
	}

	rsp, err := r.client.DeleteGlobalClusterWithResponse(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Nomad Cluster", err.Error())
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
		resp.Diagnostics.AddError("Failed To Delete Nomad Cluster", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting nomad cluster %q, got %s.", state.Name.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *NomadClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

type nomadClusterNotFoundError struct {
	name string
}

func (e *nomadClusterNotFoundError) Error() string {
	return fmt.Sprintf("nomad cluster %q not found", e.name)
}

func isNomadClusterNotFound(err error) bool {
	_, ok := err.(*nomadClusterNotFoundError)
	return ok
}

func readGlobalCluster(ctx context.Context, client *sdk.ClientWithResponses, name string) (sdk.Cluster, error) {
	rsp, err := client.GetGlobalClusterWithResponse(ctx, name)
	if err != nil {
		return sdk.Cluster{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.Cluster{}, &nomadClusterNotFoundError{name: name}
	}
	if rsp.JSON200 == nil {
		return sdk.Cluster{}, fmt.Errorf("expected 200 response when reading nomad cluster %q, got %s", name, rsp.Status())
	}

	return rsp.JSON200.Data, nil
}

func buildCreateClusterBody(plan, config NomadClusterResourceModel) (sdk.CreateGlobalClusterJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := sdk.CreateGlobalClusterJSONRequestBody{
		Name: plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}

	if !plan.Address.IsNull() && !plan.Address.IsUnknown() {
		address := plan.Address.ValueString()
		body.Address = &address
	}

	if !plan.AgentID.IsNull() && !plan.AgentID.IsUnknown() && plan.AgentID.ValueString() != "" {
		agentID, err := parseAgentID(plan.AgentID.ValueString())
		if err != nil {
			diags.AddError("Invalid Agent ID", err.Error())
			return body, diags
		}
		body.AgentId = &agentID
	}

	if !plan.ConnectivityMode.IsNull() && !plan.ConnectivityMode.IsUnknown() && plan.ConnectivityMode.ValueString() != "" {
		mode := sdk.CreateClusterRequestConnectivityMode(plan.ConnectivityMode.ValueString())
		body.ConnectivityMode = &mode
	}

	if !plan.SkipVerify.IsNull() && !plan.SkipVerify.IsUnknown() {
		skipVerify := plan.SkipVerify.ValueBool()
		body.SkipVerify = &skipVerify
	}

	if !config.AclToken.IsNull() && !config.AclToken.IsUnknown() {
		aclToken := config.AclToken.ValueString()
		body.AclToken = &aclToken
	}
	if !config.CaCert.IsNull() && !config.CaCert.IsUnknown() {
		caCert := config.CaCert.ValueString()
		body.CaCert = &caCert
	}
	if !config.TlsCert.IsNull() && !config.TlsCert.IsUnknown() {
		tlsCert := config.TlsCert.ValueString()
		body.TlsCert = &tlsCert
	}
	if !config.TlsKey.IsNull() && !config.TlsKey.IsUnknown() {
		tlsKey := config.TlsKey.ValueString()
		body.TlsKey = &tlsKey
	}

	return body, diags
}

func buildUpdateClusterBody(plan, state, config NomadClusterResourceModel) (sdk.UpdateGlobalClusterJSONRequestBody, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := sdk.UpdateGlobalClusterJSONRequestBody{}

	if stringValueChanged(plan.Description, state.Description) && !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		body.Description = &description
	}
	if stringValueChanged(plan.Address, state.Address) && !plan.Address.IsNull() && !plan.Address.IsUnknown() {
		address := plan.Address.ValueString()
		body.Address = &address
	}
	if stringValueChanged(plan.AgentID, state.AgentID) && !plan.AgentID.IsNull() && !plan.AgentID.IsUnknown() && plan.AgentID.ValueString() != "" {
		agentID, err := parseAgentID(plan.AgentID.ValueString())
		if err != nil {
			diags.AddError("Invalid Agent ID", err.Error())
			return body, diags
		}
		body.AgentId = &agentID
	}
	if stringValueChanged(plan.ConnectivityMode, state.ConnectivityMode) && !plan.ConnectivityMode.IsNull() && !plan.ConnectivityMode.IsUnknown() && plan.ConnectivityMode.ValueString() != "" {
		mode := sdk.UpdateClusterRequestConnectivityMode(plan.ConnectivityMode.ValueString())
		body.ConnectivityMode = &mode
	}
	if boolValueChanged(plan.SkipVerify, state.SkipVerify) && !plan.SkipVerify.IsNull() && !plan.SkipVerify.IsUnknown() {
		skipVerify := plan.SkipVerify.ValueBool()
		body.SkipVerify = &skipVerify
	}

	if !config.AclToken.IsNull() && !config.AclToken.IsUnknown() {
		aclToken := config.AclToken.ValueString()
		body.AclToken = &aclToken
	}
	if !config.CaCert.IsNull() && !config.CaCert.IsUnknown() {
		caCert := config.CaCert.ValueString()
		body.CaCert = &caCert
	}
	if !config.TlsCert.IsNull() && !config.TlsCert.IsUnknown() {
		tlsCert := config.TlsCert.ValueString()
		body.TlsCert = &tlsCert
	}
	if !config.TlsKey.IsNull() && !config.TlsKey.IsUnknown() {
		tlsKey := config.TlsKey.ValueString()
		body.TlsKey = &tlsKey
	}

	return body, diags
}

func stateFromNomadCluster(base NomadClusterResourceModel, cluster sdk.Cluster) NomadClusterResourceModel {
	description := base.Description
	if description.IsUnknown() {
		description = types.StringNull()
	}
	if cluster.Description != nil {
		description = types.StringValue(*cluster.Description)
	}

	address := base.Address
	if address.IsUnknown() {
		address = types.StringNull()
	}
	if cluster.Address != nil {
		address = types.StringValue(*cluster.Address)
	}

	agentID := base.AgentID
	if agentID.IsUnknown() {
		agentID = types.StringNull()
	}
	if cluster.AgentId != nil {
		agentID = types.StringValue(cluster.AgentId.String())
	}

	createdAt := types.StringNull()
	if len(cluster.Nodes) > 0 {
		_ = cluster.Nodes
	}

	updatedAt := types.StringNull()

	return NomadClusterResourceModel{
		ID:               types.StringValue(cluster.Id.String()),
		Name:             types.StringValue(cluster.Name),
		Description:      description,
		ConnectivityMode: types.StringValue(string(cluster.ConnectivityMode)),
		Address:          address,
		AgentID:          agentID,
		SkipVerify:       types.BoolValue(cluster.SkipVerify),
		AclToken:         base.AclToken,
		CaCert:           base.CaCert,
		TlsCert:          base.TlsCert,
		TlsKey:           base.TlsKey,
		Scope:            types.StringValue(cluster.Scope),
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}
