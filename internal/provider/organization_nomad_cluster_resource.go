package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &OrganizationNomadClusterResource{}
var _ resource.ResourceWithImportState = &OrganizationNomadClusterResource{}

type OrganizationNomadClusterResource struct {
	client *sdk.ClientWithResponses
}

func NewOrganizationNomadClusterResource() resource.Resource {
	return &OrganizationNomadClusterResource{}
}

type OrganizationNomadClusterResourceModel struct {
	OrgName          types.String `tfsdk:"org_name"`
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	ConnectivityMode types.String `tfsdk:"connectivity_mode"`
	Address          types.String `tfsdk:"address"`
	AgentID          types.String `tfsdk:"agent_id"`
	SkipVerify       types.Bool   `tfsdk:"skip_verify"`
	AclTokenWO       types.String `tfsdk:"acl_token_wo"`
	CaCertWO         types.String `tfsdk:"ca_cert_wo"`
	TlsCertWO        types.String `tfsdk:"tls_cert_wo"`
	TlsKeyWO         types.String `tfsdk:"tls_key_wo"`
	Scope            types.String `tfsdk:"scope"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *OrganizationNomadClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_nomad_cluster"
}

func (r *OrganizationNomadClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron organization-scoped Nomad cluster resource.",
		Attributes: map[string]schema.Attribute{
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the cluster.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
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
			"acl_token_wo": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "Nomad ACL token used in direct mode.",
			},
			"ca_cert_wo": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "PEM encoded CA certificate for direct mode.",
			},
			"tls_cert_wo": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "PEM encoded client certificate for direct mode.",
			},
			"tls_key_wo": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "PEM encoded client key for direct mode.",
			},
			"scope": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Cluster scope, typically the owning organization ID for this resource.",
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

func (r *OrganizationNomadClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationNomadClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationNomadClusterResourceModel
	var config OrganizationNomadClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization nomad cluster resource.",
		)
		return
	}

	body, diags := buildCreateClusterBody(
		NomadClusterResourceModel{
			Name:             plan.Name,
			Description:      plan.Description,
			ConnectivityMode: plan.ConnectivityMode,
			Address:          plan.Address,
			AgentID:          plan.AgentID,
			SkipVerify:       plan.SkipVerify,
		},
		NomadClusterResourceModel{
			AclTokenWO: config.AclTokenWO,
			CaCertWO:   config.CaCertWO,
			TlsCertWO:  config.TlsCertWO,
			TlsKeyWO:   config.TlsKeyWO,
		},
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.CreateOrganizationClusterWithResponse(ctx, plan.OrgName.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Organization Nomad Cluster", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create Organization Nomad Cluster", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Organization Nomad Cluster Already Exists", formatAPIError(rsp.JSON409))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Create Organization Nomad Cluster", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when creating organization nomad cluster, got %s.", rsp.Status()),
		)
		return
	}

	state := stateFromOrganizationNomadCluster(plan, rsp.JSON201.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationNomadClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationNomadClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization nomad cluster resource.",
		)
		return
	}

	cluster, err := readOrganizationCluster(ctx, r.client, state.OrgName.ValueString(), state.Name.ValueString())
	if err != nil {
		if isOrganizationNomadClusterNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Organization Nomad Cluster", err.Error())
		return
	}

	newState := stateFromOrganizationNomadCluster(state, cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OrganizationNomadClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationNomadClusterResourceModel
	var state OrganizationNomadClusterResourceModel
	var config OrganizationNomadClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization nomad cluster resource.",
		)
		return
	}

	body, diags := buildUpdateClusterBody(
		NomadClusterResourceModel{
			Description:      plan.Description,
			ConnectivityMode: plan.ConnectivityMode,
			Address:          plan.Address,
			AgentID:          plan.AgentID,
			SkipVerify:       plan.SkipVerify,
		},
		NomadClusterResourceModel{
			Description:      state.Description,
			ConnectivityMode: state.ConnectivityMode,
			Address:          state.Address,
			AgentID:          state.AgentID,
			SkipVerify:       state.SkipVerify,
		},
		NomadClusterResourceModel{
			AclTokenWO: config.AclTokenWO,
			CaCertWO:   config.CaCertWO,
			TlsCertWO:  config.TlsCertWO,
			TlsKeyWO:   config.TlsKeyWO,
		},
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rsp, err := r.client.UpdateOrganizationClusterWithResponse(ctx, state.OrgName.ValueString(), state.Name.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Update Organization Nomad Cluster", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Update Organization Nomad Cluster", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Update Organization Nomad Cluster", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when updating organization nomad cluster %q in org %q, got %s.", state.Name.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	newState := stateFromOrganizationNomadCluster(plan, rsp.JSON200.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OrganizationNomadClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationNomadClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization nomad cluster resource.",
		)
		return
	}

	rsp, err := r.client.DeleteOrganizationClusterWithResponse(ctx, state.OrgName.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete Organization Nomad Cluster", err.Error())
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
		resp.Diagnostics.AddError("Failed To Delete Organization Nomad Cluster", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200/204 response when deleting organization nomad cluster %q in org %q, got %s.", state.Name.ValueString(), state.OrgName.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *OrganizationNomadClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Expected import identifier in the format `org_name/cluster_name`.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

type organizationNomadClusterNotFoundError struct {
	orgName string
	name    string
}

func (e *organizationNomadClusterNotFoundError) Error() string {
	return fmt.Sprintf("organization nomad cluster %q in org %q not found", e.name, e.orgName)
}

func isOrganizationNomadClusterNotFound(err error) bool {
	_, ok := err.(*organizationNomadClusterNotFoundError)
	return ok
}

func readOrganizationCluster(ctx context.Context, client *sdk.ClientWithResponses, orgName, name string) (sdk.Cluster, error) {
	rsp, err := client.ListOrganizationClustersWithResponse(ctx, orgName, nil)
	if err != nil {
		return sdk.Cluster{}, err
	}

	if rsp.JSON400 != nil {
		return sdk.Cluster{}, fmt.Errorf("failed to list organization nomad clusters in org %q: %s", orgName, formatAPIError(rsp.JSON400))
	}
	if rsp.JSON401 != nil {
		return sdk.Cluster{}, fmt.Errorf("unauthorized reading organization nomad clusters in org %q: %s", orgName, formatAPIError(rsp.JSON401))
	}
	if rsp.JSON403 != nil {
		return sdk.Cluster{}, fmt.Errorf("forbidden reading organization nomad clusters in org %q: %s", orgName, formatAPIError(rsp.JSON403))
	}
	if rsp.JSON404 != nil {
		return sdk.Cluster{}, &organizationNomadClusterNotFoundError{orgName: orgName, name: name}
	}
	if rsp.JSON500 != nil {
		return sdk.Cluster{}, fmt.Errorf("failed to list organization nomad clusters in org %q: %s", orgName, formatAPIError(rsp.JSON500))
	}
	if rsp.JSON200 == nil {
		return sdk.Cluster{}, fmt.Errorf("expected 200 response when listing organization nomad clusters in org %q, got %s", orgName, rsp.Status())
	}

	for _, cluster := range rsp.JSON200.Data {
		if cluster.Name == name {
			return cluster, nil
		}
	}

	return sdk.Cluster{}, &organizationNomadClusterNotFoundError{orgName: orgName, name: name}
}

func stateFromOrganizationNomadCluster(base OrganizationNomadClusterResourceModel, cluster sdk.Cluster) OrganizationNomadClusterResourceModel {
	state := stateFromNomadCluster(
		NomadClusterResourceModel{
			ID:               base.ID,
			Name:             base.Name,
			Description:      base.Description,
			ConnectivityMode: base.ConnectivityMode,
			Address:          base.Address,
			AgentID:          base.AgentID,
			SkipVerify:       base.SkipVerify,
			Scope:            base.Scope,
			CreatedAt:        base.CreatedAt,
			UpdatedAt:        base.UpdatedAt,
		},
		cluster,
	)

	return OrganizationNomadClusterResourceModel{
		OrgName:          base.OrgName,
		ID:               state.ID,
		Name:             state.Name,
		Description:      state.Description,
		ConnectivityMode: state.ConnectivityMode,
		Address:          state.Address,
		AgentID:          state.AgentID,
		SkipVerify:       state.SkipVerify,
		Scope:            state.Scope,
		CreatedAt:        state.CreatedAt,
		UpdatedAt:        state.UpdatedAt,
	}
}
