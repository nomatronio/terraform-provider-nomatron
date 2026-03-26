package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &OrganizationMemberResource{}
var _ resource.ResourceWithImportState = &OrganizationMemberResource{}

type OrganizationMemberResource struct {
	client *sdk.ClientWithResponses
}

func NewOrganizationMemberResource() resource.Resource {
	return &OrganizationMemberResource{}
}

type OrganizationMemberResourceModel struct {
	ID       types.String `tfsdk:"id"`
	OrgName  types.String `tfsdk:"org_name"`
	Username types.String `tfsdk:"username"`
	UserID   types.String `tfsdk:"user_id"`
	JoinedAt types.String `tfsdk:"joined_at"`
}

func (r *OrganizationMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_member"
}

func (r *OrganizationMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron organization member resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable composite identifier for the organization membership.",
			},
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Username to add to the organization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User ID for the organization member.",
			},
			"joined_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the user joined the organization.",
			},
		},
	}
}

func (r *OrganizationMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationMemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization member resource.",
		)
		return
	}

	body := sdk.AddOrganizationMemberJSONRequestBody{
		Username: plan.Username.ValueString(),
	}

	rsp, err := r.client.AddOrganizationMemberWithResponse(ctx, plan.OrgName.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Add Organization Member", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Add Organization Member", formatAPIError(rsp.JSON400))
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
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when adding member %q to organization %q, got %s.", plan.Username.ValueString(), plan.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	member, err := findOrganizationMemberByUsername(rsp.JSON200.Data.Items, plan.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed To Add Organization Member", err.Error())
		return
	}

	state := stateFromOrganizationMember(plan, member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization member resource.",
		)
		return
	}

	orgName, username, err := parseOrganizationMemberID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Organization Member ID", err.Error())
		return
	}

	member, err := readOrganizationMember(ctx, r.client, orgName, username)
	if err != nil {
		if isOrganizationMemberNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Organization Member", err.Error())
		return
	}

	newState := stateFromOrganizationMember(state, member)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *OrganizationMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unsupported Update",
		"Organization members use replace semantics. Terraform should create a new membership and delete the old one instead of updating in place.",
	)
}

func (r *OrganizationMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the organization member resource.",
		)
		return
	}

	orgName, username, err := parseOrganizationMemberID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Organization Member ID", err.Error())
		return
	}

	rsp, err := r.client.RemoveOrganizationMemberWithResponse(ctx, orgName, username)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Remove Organization Member", err.Error())
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
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Remove Organization Member", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when removing member %q from organization %q, got %s.", username, orgName, rsp.Status()),
		)
		return
	}
}

func (r *OrganizationMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgName, username, err := parseOrganizationMemberID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Organization Member ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
}

type organizationMemberNotFoundError struct {
	orgName  string
	username string
}

func (e *organizationMemberNotFoundError) Error() string {
	return fmt.Sprintf("organization member %q in %q not found", e.username, e.orgName)
}

func isOrganizationMemberNotFound(err error) bool {
	_, ok := err.(*organizationMemberNotFoundError)
	return ok
}

func readOrganizationMember(ctx context.Context, client *sdk.ClientWithResponses, orgName, username string) (sdk.OrganizationMember, error) {
	rsp, err := client.ListOrganizationMembersWithResponse(ctx, orgName, nil)
	if err != nil {
		return sdk.OrganizationMember{}, err
	}

	if rsp.JSON404 != nil {
		return sdk.OrganizationMember{}, &organizationMemberNotFoundError{orgName: orgName, username: username}
	}
	if rsp.JSON200 == nil {
		return sdk.OrganizationMember{}, fmt.Errorf("expected 200 response when listing members for organization %q, got %s", orgName, rsp.Status())
	}

	member, err := findOrganizationMemberByUsername(rsp.JSON200.Data.Items, username)
	if err != nil {
		return sdk.OrganizationMember{}, &organizationMemberNotFoundError{orgName: orgName, username: username}
	}

	return member, nil
}

func findOrganizationMemberByUsername(items []sdk.OrganizationMember, username string) (sdk.OrganizationMember, error) {
	for _, member := range items {
		if member.Username == username {
			return member, nil
		}
	}

	return sdk.OrganizationMember{}, fmt.Errorf("no organization member was found with username %q", username)
}

func stateFromOrganizationMember(base OrganizationMemberResourceModel, member sdk.OrganizationMember) OrganizationMemberResourceModel {
	id := base.ID
	if !base.OrgName.IsNull() && !base.OrgName.IsUnknown() && !base.Username.IsNull() && !base.Username.IsUnknown() {
		id = types.StringValue(buildOrganizationMemberID(base.OrgName.ValueString(), base.Username.ValueString()))
	}

	joinedAt := base.JoinedAt
	if joinedAt.IsUnknown() {
		joinedAt = types.StringNull()
	}
	if !member.JoinedAt.IsZero() {
		joinedAt = types.StringValue(member.JoinedAt.Format(time.RFC3339))
	}

	userID := base.UserID
	if userID.IsUnknown() {
		userID = types.StringNull()
	}
	if member.UserId.String() != "00000000-0000-0000-0000-000000000000" {
		userID = types.StringValue(member.UserId.String())
	}

	username := base.Username
	if username.IsUnknown() {
		username = types.StringNull()
	}
	if member.Username != "" {
		username = types.StringValue(member.Username)
	}

	return OrganizationMemberResourceModel{
		ID:       id,
		OrgName:  base.OrgName,
		Username: username,
		UserID:   userID,
		JoinedAt: joinedAt,
	}
}

func buildOrganizationMemberID(orgName, username string) string {
	values := url.Values{}
	values.Set("org_name", orgName)
	values.Set("username", username)
	return values.Encode()
}

func parseOrganizationMemberID(raw string) (orgName, username string, err error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid organization member id %q: %w", raw, err)
	}

	orgName = values.Get("org_name")
	username = values.Get("username")

	if orgName == "" || username == "" {
		return "", "", fmt.Errorf("invalid organization member id %q: expected org_name and username query values", raw)
	}

	return orgName, username, nil
}
