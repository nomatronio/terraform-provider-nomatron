package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ resource.Resource = &GroupMemberResource{}
var _ resource.ResourceWithImportState = &GroupMemberResource{}

type GroupMemberResource struct {
	client *sdk.ClientWithResponses
}

func NewGroupMemberResource() resource.Resource {
	return &GroupMemberResource{}
}

type GroupMemberResourceModel struct {
	ID        types.String `tfsdk:"id"`
	OrgName   types.String `tfsdk:"org_name"`
	GroupName types.String `tfsdk:"group_name"`
	Username  types.String `tfsdk:"username"`
}

func (r *GroupMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_member"
}

func (r *GroupMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron group member resource. Note: the current group-member endpoints expose membership presence by username, but do not return per-member metadata like joined timestamps.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable composite identifier for the group membership.",
			},
			"org_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Organization name that owns the group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Group name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Username to add to the group.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *GroupMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupMemberResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group member resource.",
		)
		return
	}

	rsp, err := r.client.PutOrganizationsOrgSlugGroupsGroupNameMembersUsernameWithResponse(
		ctx,
		plan.OrgName.ValueString(),
		plan.GroupName.ValueString(),
		plan.Username.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Add Group Member", err.Error())
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
		resp.Diagnostics.AddError("Group Not Found", formatAPIError(rsp.JSON404))
		return
	}
	if rsp.JSON500 != nil {
		resp.Diagnostics.AddError("Failed To Add Group Member", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when adding member %q to group %q in organization %q, got %s.", plan.Username.ValueString(), plan.GroupName.ValueString(), plan.OrgName.ValueString(), rsp.Status()),
		)
		return
	}

	plan.ID = types.StringValue(buildGroupMemberID(plan.OrgName.ValueString(), plan.GroupName.ValueString(), plan.Username.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group member resource.",
		)
		return
	}

	orgName, groupName, username, err := parseGroupMemberID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group Member ID", err.Error())
		return
	}

	exists, err := groupMemberExists(ctx, r.client, orgName, groupName, username)
	if err != nil {
		if isGroupMemberNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Group Member", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unsupported Update",
		"Group members use replace semantics. Terraform should create a new membership and delete the old one instead of updating in place.",
	)
}

func (r *GroupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupMemberResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the group member resource.",
		)
		return
	}

	orgName, groupName, username, err := parseGroupMemberID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group Member ID", err.Error())
		return
	}

	rsp, err := r.client.DeleteOrganizationsOrgSlugGroupsGroupNameMembersUsernameWithResponse(ctx, orgName, groupName, username)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Remove Group Member", err.Error())
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
		resp.Diagnostics.AddError("Failed To Remove Group Member", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when removing member %q from group %q in organization %q, got %s.", username, groupName, orgName, rsp.Status()),
		)
		return
	}
}

func (r *GroupMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	orgName, groupName, username, err := parseGroupMemberID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Group Member ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_name"), groupName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
}

type groupMemberNotFoundError struct {
	orgName   string
	groupName string
	username  string
}

func (e *groupMemberNotFoundError) Error() string {
	return fmt.Sprintf("group member %q in group %q/%q not found", e.username, e.orgName, e.groupName)
}

func isGroupMemberNotFound(err error) bool {
	_, ok := err.(*groupMemberNotFoundError)
	return ok
}

func groupMemberExists(ctx context.Context, client *sdk.ClientWithResponses, orgName, groupName, username string) (bool, error) {
	rsp, err := client.GetOrganizationsOrgSlugGroupsGroupNameMembersWithResponse(ctx, orgName, groupName)
	if err != nil {
		return false, err
	}

	if rsp.JSON404 != nil {
		return false, &groupMemberNotFoundError{orgName: orgName, groupName: groupName, username: username}
	}
	if rsp.JSON200 == nil {
		return false, fmt.Errorf("expected 200 response when listing members for group %q in organization %q, got %s", groupName, orgName, rsp.Status())
	}

	for _, got := range rsp.JSON200.Data {
		if got == username {
			return true, nil
		}
	}

	return false, &groupMemberNotFoundError{orgName: orgName, groupName: groupName, username: username}
}

func buildGroupMemberID(orgName, groupName, username string) string {
	values := url.Values{}
	values.Set("org_name", orgName)
	values.Set("group_name", groupName)
	values.Set("username", username)
	return values.Encode()
}

func parseGroupMemberID(raw string) (orgName, groupName, username string, err error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid group member id %q: %w", raw, err)
	}

	orgName = values.Get("org_name")
	groupName = values.Get("group_name")
	username = values.Get("username")

	if orgName == "" || groupName == "" || username == "" {
		return "", "", "", fmt.Errorf("invalid group member id %q: expected org_name, group_name, and username query values", raw)
	}

	return orgName, groupName, username, nil
}
