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

var _ resource.Resource = &RoleAssignmentResource{}
var _ resource.ResourceWithImportState = &RoleAssignmentResource{}

type RoleAssignmentResource struct {
	client *sdk.ClientWithResponses
}

func NewRoleAssignmentResource() resource.Resource {
	return &RoleAssignmentResource{}
}

type RoleAssignmentResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Subject types.String `tfsdk:"subject"`
	Role    types.String `tfsdk:"role"`
	Domain  types.String `tfsdk:"domain"`
}

func (r *RoleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_assignment"
}

func (r *RoleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron role assignment resource. Note: the current SDK does not expose a role-assignment read/list endpoint, so Terraform cannot verify remote drift for this resource during refresh.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable composite identifier for the role assignment.",
			},
			"subject": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Assignment subject, typically in the form user:<uuid> or service-account:<uuid>.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Role name to assign.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Role domain or scope, such as global or organization:<uuid>.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *RoleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleAssignmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the role assignment resource.",
		)
		return
	}

	body := sdk.AssignRoleJSONRequestBody{
		Subject: plan.Subject.ValueString(),
		Role:    plan.Role.ValueString(),
		Domain:  plan.Domain.ValueString(),
	}

	rsp, err := r.client.AssignRoleWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Assign Role", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Assign Role", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Assign Role", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.JSON201 == nil {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 201 response when assigning role, got %s.", rsp.Status()),
		)
		return
	}

	plan.ID = types.StringValue(buildRoleAssignmentID(plan.Subject.ValueString(), plan.Role.ValueString(), plan.Domain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, _, _, err := parseRoleAssignmentID(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid Role Assignment ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unsupported Update",
		"Role assignments use replace semantics. Terraform should create a new assignment and delete the old one instead of updating in place.",
	)
}

func (r *RoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleAssignmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the role assignment resource.",
		)
		return
	}

	subject, role, domain, err := parseRoleAssignmentID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Role Assignment ID", err.Error())
		return
	}

	body := sdk.RemoveRoleJSONRequestBody{
		Subject: subject,
		Role:    role,
		Domain:  domain,
	}

	rsp, err := r.client.RemoveRoleWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Remove Role Assignment", err.Error())
		return
	}

	if rsp.JSON400 != nil {
		resp.Diagnostics.AddError("Failed To Remove Role Assignment", formatAPIError(rsp.JSON400))
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
		resp.Diagnostics.AddError("Failed To Remove Role Assignment", formatAPIError(rsp.JSON500))
		return
	}
	if rsp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API Response",
			fmt.Sprintf("Expected 200 response when removing role assignment %q, got %s.", state.ID.ValueString(), rsp.Status()),
		)
		return
	}
}

func (r *RoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	subject, role, domain, err := parseRoleAssignmentID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Role Assignment ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subject"), subject)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role"), role)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), domain)...)
}

func buildRoleAssignmentID(subject, role, domain string) string {
	values := url.Values{}
	values.Set("subject", subject)
	values.Set("role", role)
	values.Set("domain", domain)
	return values.Encode()
}

func parseRoleAssignmentID(raw string) (subject, role, domain string, err error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid role assignment id %q: %w", raw, err)
	}

	subject = values.Get("subject")
	role = values.Get("role")
	domain = values.Get("domain")

	if subject == "" || role == "" || domain == "" {
		return "", "", "", fmt.Errorf("invalid role assignment id %q: expected subject, role, and domain query values", raw)
	}

	return subject, role, domain, nil
}
