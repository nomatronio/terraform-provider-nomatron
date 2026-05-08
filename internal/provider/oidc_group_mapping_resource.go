package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ resource.Resource = &OIDCGroupMappingResource{}
var _ resource.ResourceWithImportState = &OIDCGroupMappingResource{}

type OIDCGroupMappingResource struct {
	client *sdk.ClientWithResponses
}

func NewOIDCGroupMappingResource() resource.Resource {
	return &OIDCGroupMappingResource{}
}

type OIDCGroupMappingResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProviderSlug  types.String `tfsdk:"provider_slug"`
	ProviderID    types.String `tfsdk:"provider_id"`
	ExternalGroup types.String `tfsdk:"external_group"`
	Role          types.String `tfsdk:"role"`
	Domain        types.String `tfsdk:"domain"`
	Description   types.String `tfsdk:"description"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

func (r *OIDCGroupMappingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_group_mapping"
}

func (r *OIDCGroupMappingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron OIDC external-group to RBAC role mapping resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OIDC group mapping ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provider_slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OIDC provider slug that owns this mapping.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OIDC provider ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_group": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "External OIDC group claim value.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Nomatron role name assigned to matching users.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "RBAC domain, for example global or an organization domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Mapping description.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this mapping is active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the mapping was created.",
			},
		},
	}
}

func (r *OIDCGroupMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OIDCGroupMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OIDCGroupMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC group mapping resource.")
		return
	}

	body := sdk.CreateOIDCGroupMappingJSONRequestBody{
		ExternalGroup: plan.ExternalGroup.ValueString(),
		Role:          plan.Role.ValueString(),
		Domain:        plan.Domain.ValueString(),
		Description:   terraformOptionalString(plan.Description),
		Enabled:       terraformOptionalBool(plan.Enabled),
	}

	rsp, err := r.client.CreateOIDCGroupMappingWithResponse(ctx, plan.ProviderSlug.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create OIDC Group Mapping", err.Error())
		return
	}
	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Create OIDC Group Mapping", formatAPIError(rsp.JSON400))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError("Unexpected API Response", fmt.Sprintf("Expected 201 response when creating OIDC group mapping for provider %q, got %s.", plan.ProviderSlug.ValueString(), rsp.Status()))
		return
	}

	state := stateFromOIDCGroupMapping(plan.ProviderSlug.ValueString(), rsp.JSON201.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OIDCGroupMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OIDCGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC group mapping resource.")
		return
	}

	mappingID, err := parseAgentID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid OIDC Group Mapping ID", err.Error())
		return
	}

	mapping, err := readOIDCGroupMapping(ctx, r.client, state.ProviderSlug.ValueString(), mappingID)
	if err != nil {
		if isOIDCGroupMappingNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read OIDC Group Mapping", err.Error())
		return
	}

	newState := stateFromOIDCGroupMapping(state.ProviderSlug.ValueString(), mapping)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OIDCGroupMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"OIDC Group Mapping Update Is Not Supported",
		"Nomatron does not expose an update endpoint for OIDC group mappings. Terraform will replace this resource when configurable attributes change.",
	)
}

func (r *OIDCGroupMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OIDCGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the OIDC group mapping resource.")
		return
	}

	mappingID, err := parseAgentID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid OIDC Group Mapping ID", err.Error())
		return
	}

	rsp, err := r.client.DeleteOIDCGroupMappingWithResponse(ctx, mappingID)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Delete OIDC Group Mapping", err.Error())
		return
	}
	if rsp.JSON404 != nil {
		return
	}
	if rsp.StatusCode() != http.StatusOK && rsp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError("Unexpected API Response", fmt.Sprintf("Expected 200/204 response when deleting OIDC group mapping %q, got %s.", state.ID.ValueString(), rsp.Status()))
		return
	}
}

func (r *OIDCGroupMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	provider, mappingID, err := parseOIDCGroupMappingImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_slug"), provider)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), mappingID)...)
}

type oidcGroupMappingNotFoundError struct {
	id string
}

func (e *oidcGroupMappingNotFoundError) Error() string {
	return fmt.Sprintf("OIDC group mapping %q not found", e.id)
}

func isOIDCGroupMappingNotFound(err error) bool {
	_, ok := err.(*oidcGroupMappingNotFoundError)
	return ok
}

func readOIDCGroupMapping(ctx context.Context, client *sdk.ClientWithResponses, provider string, mappingID openapi_types.UUID) (sdk.OIDCGroupRoleMapping, error) {
	mappings, err := listOIDCGroupMappings(ctx, client, provider)
	if err != nil {
		return sdk.OIDCGroupRoleMapping{}, err
	}

	for _, mapping := range mappings {
		if mapping.Id == mappingID {
			return mapping, nil
		}
	}

	return sdk.OIDCGroupRoleMapping{}, &oidcGroupMappingNotFoundError{id: mappingID.String()}
}

func listOIDCGroupMappings(ctx context.Context, client *sdk.ClientWithResponses, provider string) ([]sdk.OIDCGroupRoleMapping, error) {
	rsp, err := client.ListOIDCGroupMappingsWithResponse(ctx, provider)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode() == http.StatusNotFound {
		return nil, &oidcProviderNotFoundError{slug: provider}
	}
	if rsp.JSON200 == nil {
		return nil, fmt.Errorf("expected 200 response when listing OIDC group mappings for provider %q, got %s", provider, rsp.Status())
	}
	return rsp.JSON200.Data, nil
}

func stateFromOIDCGroupMapping(provider string, mapping sdk.OIDCGroupRoleMapping) OIDCGroupMappingResourceModel {
	description := types.StringNull()
	if mapping.Description != nil {
		description = types.StringValue(*mapping.Description)
	}

	createdAt := types.StringNull()
	if !mapping.CreatedAt.IsZero() {
		createdAt = types.StringValue(mapping.CreatedAt.Format(time.RFC3339))
	}

	return OIDCGroupMappingResourceModel{
		ID:            types.StringValue(mapping.Id.String()),
		ProviderSlug:  types.StringValue(provider),
		ProviderID:    types.StringValue(mapping.ProviderId.String()),
		ExternalGroup: types.StringValue(mapping.ExternalGroup),
		Role:          types.StringValue(mapping.Role),
		Domain:        types.StringValue(mapping.Domain),
		Description:   description,
		Enabled:       types.BoolValue(mapping.Enabled),
		CreatedAt:     createdAt,
	}
}

func parseOIDCGroupMappingImportID(raw string) (string, string, error) {
	parts := strings.Split(raw, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("expected import id in the form <provider_slug>/<mapping_id>")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func findOIDCGroupMapping(ctx context.Context, client *sdk.ClientWithResponses, provider, externalGroup, role, domain string) (sdk.OIDCGroupRoleMapping, error) {
	mappings, err := listOIDCGroupMappings(ctx, client, provider)
	if err != nil {
		return sdk.OIDCGroupRoleMapping{}, err
	}

	var matches []sdk.OIDCGroupRoleMapping
	for _, mapping := range mappings {
		if mapping.ExternalGroup == externalGroup && mapping.Role == role && mapping.Domain == domain {
			matches = append(matches, mapping)
		}
	}

	switch len(matches) {
	case 0:
		return sdk.OIDCGroupRoleMapping{}, fmt.Errorf("no OIDC group mapping was found for provider %q, external_group %q, role %q, and domain %q", provider, externalGroup, role, domain)
	case 1:
		return matches[0], nil
	default:
		return sdk.OIDCGroupRoleMapping{}, fmt.Errorf("multiple OIDC group mappings were found for provider %q, external_group %q, role %q, and domain %q", provider, externalGroup, role, domain)
	}
}
