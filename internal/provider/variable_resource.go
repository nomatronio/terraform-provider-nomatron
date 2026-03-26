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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	variableScopeGlobal       = "global"
	variableScopeOrganization = "organization"
	variableScopeApp          = "app"
	variableScopeJob          = "job"

	variableAPIScopeGlobal = "global"
	variableAPIScopeOrg    = "org"
	variableAPIScopeApp    = "app"
	variableAPIScopeAppJob = "app_job"

	variableSensitivityNormal    = "normal"
	variableSensitivitySensitive = "sensitive"
)

var _ resource.Resource = &VariableResource{}
var _ resource.ResourceWithConfigure = &VariableResource{}
var _ resource.ResourceWithImportState = &VariableResource{}
var _ resource.ResourceWithValidateConfig = &VariableResource{}

type VariableResource struct {
	client *sdk.ClientWithResponses
}

func NewVariableResource() resource.Resource {
	return &VariableResource{}
}

type VariableResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Scope       types.String `tfsdk:"scope"`
	OrgName     types.String `tfsdk:"org_name"`
	AppSlug     types.String `tfsdk:"app_slug"`
	JobSlug     types.String `tfsdk:"job_slug"`
	Key         types.String `tfsdk:"key"`
	Description types.String `tfsdk:"description"`
	Sensitive   types.Bool   `tfsdk:"sensitive"`
	ValueType   types.String `tfsdk:"value_type"`
	Value       types.String `tfsdk:"value"`
	ValueWO     types.String `tfsdk:"value_wo"`
	ValueID     types.String `tfsdk:"value_id"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *VariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

func (r *VariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Nomatron variable resource supporting global, organization, application, and job scopes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Variable ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scope": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Variable scope. One of `global`, `organization`, `app`, or `job`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Organization name for organization, app, or job scoped variables.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_slug": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Application slug for app or job scoped variables.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"job_slug": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Job slug for job scoped variables.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Variable key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional variable description.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sensitive": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the variable value is sensitive.",
			},
			"value_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("string"),
				MarkdownDescription: "Variable value type. One of `string`, `number`, `bool`, or `json`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"value": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Stored default value for non-sensitive variables.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"value_wo": schema.StringAttribute{
				Optional:            true,
				WriteOnly:           true,
				Sensitive:           true,
				MarkdownDescription: "Write-only default value for sensitive variables.",
			},
			"value_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Default value row ID, when present.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the variable was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the variable was last updated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VariableResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data VariableResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Scope.IsUnknown() || data.Scope.IsNull() {
		return
	}

	scope := normalizeVariableScope(data.Scope.ValueString())
	switch scope {
	case variableScopeGlobal:
		if attributeIsSet(data.OrgName) {
			resp.Diagnostics.AddAttributeError(path.Root("org_name"), "Invalid Scope Configuration", "`org_name` must not be set for global variables.")
		}
		if attributeIsSet(data.AppSlug) {
			resp.Diagnostics.AddAttributeError(path.Root("app_slug"), "Invalid Scope Configuration", "`app_slug` must not be set for global variables.")
		}
		if attributeIsSet(data.JobSlug) {
			resp.Diagnostics.AddAttributeError(path.Root("job_slug"), "Invalid Scope Configuration", "`job_slug` must not be set for global variables.")
		}
	case variableScopeOrganization:
		if !attributeIsSet(data.OrgName) {
			resp.Diagnostics.AddAttributeError(path.Root("org_name"), "Missing Scope Attribute", "`org_name` is required for organization scoped variables.")
		}
		if attributeIsSet(data.AppSlug) {
			resp.Diagnostics.AddAttributeError(path.Root("app_slug"), "Invalid Scope Configuration", "`app_slug` must not be set for organization scoped variables.")
		}
		if attributeIsSet(data.JobSlug) {
			resp.Diagnostics.AddAttributeError(path.Root("job_slug"), "Invalid Scope Configuration", "`job_slug` must not be set for organization scoped variables.")
		}
	case variableScopeApp:
		if !attributeIsSet(data.OrgName) {
			resp.Diagnostics.AddAttributeError(path.Root("org_name"), "Missing Scope Attribute", "`org_name` is required for app scoped variables.")
		}
		if !attributeIsSet(data.AppSlug) {
			resp.Diagnostics.AddAttributeError(path.Root("app_slug"), "Missing Scope Attribute", "`app_slug` is required for app scoped variables.")
		}
		if attributeIsSet(data.JobSlug) {
			resp.Diagnostics.AddAttributeError(path.Root("job_slug"), "Invalid Scope Configuration", "`job_slug` must not be set for app scoped variables.")
		}
	case variableScopeJob:
		if !attributeIsSet(data.OrgName) {
			resp.Diagnostics.AddAttributeError(path.Root("org_name"), "Missing Scope Attribute", "`org_name` is required for job scoped variables.")
		}
		if !attributeIsSet(data.AppSlug) {
			resp.Diagnostics.AddAttributeError(path.Root("app_slug"), "Missing Scope Attribute", "`app_slug` is required for job scoped variables.")
		}
		if !attributeIsSet(data.JobSlug) {
			resp.Diagnostics.AddAttributeError(path.Root("job_slug"), "Missing Scope Attribute", "`job_slug` is required for job scoped variables.")
		}
	default:
		resp.Diagnostics.AddAttributeError(path.Root("scope"), "Invalid Scope", "`scope` must be one of `global`, `organization`, `app`, or `job`.")
	}

	if attributeIsSet(data.Value) && attributeIsSet(data.ValueWO) {
		resp.Diagnostics.AddAttributeError(path.Root("value"), "Invalid Value Configuration", "Only one of `value` or `value_wo` can be set.")
		resp.Diagnostics.AddAttributeError(path.Root("value_wo"), "Invalid Value Configuration", "Only one of `value` or `value_wo` can be set.")
	}

	if !data.ValueType.IsUnknown() && !data.ValueType.IsNull() {
		switch strings.ToLower(strings.TrimSpace(data.ValueType.ValueString())) {
		case "string", "number", "bool", "json":
		default:
			resp.Diagnostics.AddAttributeError(path.Root("value_type"), "Invalid Value Type", "`value_type` must be one of `string`, `number`, `bool`, or `json`.")
		}
	}

	sensitive := false
	if !data.Sensitive.IsUnknown() && !data.Sensitive.IsNull() {
		sensitive = data.Sensitive.ValueBool()
	}

	if sensitive {
		if attributeIsSet(data.Value) {
			resp.Diagnostics.AddAttributeError(path.Root("value"), "Invalid Sensitive Value Configuration", "Sensitive variables must use `value_wo` instead of `value` so the secret is not stored in Terraform state.")
		}
		if attributeIsSet(data.ValueWO) && !req.ClientCapabilities.WriteOnlyAttributesAllowed {
			resp.Diagnostics.AddAttributeError(path.Root("value_wo"), "Unsupported Terraform Version", "This Terraform client does not support write-only resource attributes. Upgrade Terraform to use `value_wo`.")
		}
	} else if attributeIsSet(data.ValueWO) {
		resp.Diagnostics.AddAttributeError(path.Root("value_wo"), "Invalid Non-Sensitive Value Configuration", "Non-sensitive variables must use `value` so the default value is tracked in Terraform state.")
	}
}

func (r *VariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VariableResourceModel
	var config VariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the variable resource.")
		return
	}

	body := sdk.CreateVariableRequest{
		Key: plan.Key.ValueString(),
	}

	if attributeIsSet(plan.Description) {
		description := plan.Description.ValueString()
		body.Description = &description
	}
	sensitivity := variableSensitivityFromBool(plan.Sensitive)
	body.Sensitivity = &sensitivity
	if attributeIsSet(plan.ValueType) {
		valueType := strings.ToLower(plan.ValueType.ValueString())
		body.ValueType = &valueType
	}

	variableRecord, err := createVariableByScope(ctx, r.client, plan, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Create Variable", err.Error())
		return
	}

	defaultValue, hasDefault := configuredVariableValue(plan, config)
	var defaultRow *sdk.VariableValue
	if hasDefault {
		defaultRow, err = upsertVariableDefaultValueByScope(ctx, r.client, plan, variableRecord.Id, defaultValue)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Set Variable Default Value", err.Error())
			return
		}
	}

	state := stateFromVariable(plan, variableRecord, defaultRow)
	if hasDefault && !plan.Sensitive.ValueBool() {
		state.Value = types.StringValue(defaultValue)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *VariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state VariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the variable resource.")
		return
	}

	variableID, err := parseVariableUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Variable ID", err.Error())
		return
	}

	variableRecord, err := getVariableByScope(ctx, r.client, state, variableID)
	if err != nil {
		if isVariableNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Read Variable", err.Error())
		return
	}

	defaultRow, err := getVariableDefaultValueByScope(ctx, r.client, state, variableID)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Variable Default Value", err.Error())
		return
	}

	newState := stateFromVariable(state, variableRecord, defaultRow)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *VariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VariableResourceModel
	var state VariableResourceModel
	var config VariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the variable resource.")
		return
	}

	variableID, err := parseVariableUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Variable ID", err.Error())
		return
	}

	body := sdk.UpdateVariableRequest{}
	if stringValueChanged(plan.Description, state.Description) {
		if attributeIsSet(plan.Description) {
			description := plan.Description.ValueString()
			body.Description = &description
		} else {
			description := ""
			body.Description = &description
		}
	}
	if boolValueChanged(plan.Sensitive, state.Sensitive) {
		sensitivity := variableSensitivityFromBool(plan.Sensitive)
		body.Sensitivity = &sensitivity
	}
	if stringValueChanged(plan.ValueType, state.ValueType) {
		valueType := strings.ToLower(plan.ValueType.ValueString())
		body.ValueType = &valueType
	}

	variableRecord, err := updateVariableByScope(ctx, r.client, plan, variableID, body)
	if err != nil {
		if isVariableNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed To Update Variable", err.Error())
		return
	}

	defaultValue, hasDefault := configuredVariableValue(plan, config)
	if hasDefault {
		if _, err := upsertVariableDefaultValueByScope(ctx, r.client, plan, variableID, defaultValue); err != nil {
			resp.Diagnostics.AddError("Failed To Update Variable Default Value", err.Error())
			return
		}
	}

	defaultRow, err := getVariableDefaultValueByScope(ctx, r.client, plan, variableID)
	if err != nil {
		resp.Diagnostics.AddError("Failed To Read Variable Default Value", err.Error())
		return
	}

	newState := stateFromVariable(plan, variableRecord, defaultRow)
	if hasDefault && !plan.Sensitive.ValueBool() {
		newState.Value = types.StringValue(defaultValue)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *VariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Not Configured", "The provider client was not configured for the variable resource.")
		return
	}

	variableID, err := parseVariableUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Variable ID", err.Error())
		return
	}

	if err := deleteVariableByScope(ctx, r.client, state, variableID); err != nil {
		if isVariableNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed To Delete Variable", err.Error())
	}
}

func (r *VariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	scope, orgName, appSlug, jobSlug, variableID, err := parseVariableImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("scope"), scope)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_name"), orgName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_slug"), appSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("job_slug"), jobSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), variableID)...)
}

type variableNotFoundError struct {
	scope string
	id    string
}

func (e *variableNotFoundError) Error() string {
	return fmt.Sprintf("variable %q in scope %q not found", e.id, e.scope)
}

func isVariableNotFound(err error) bool {
	_, ok := err.(*variableNotFoundError)
	return ok
}

func parseVariableUUID(raw string) (openapi_types.UUID, error) {
	return parseAgentID(raw)
}

func normalizeVariableScope(raw string) string {
	switch normalized := strings.ToLower(strings.TrimSpace(raw)); normalized {
	case variableScopeGlobal:
		return variableScopeGlobal
	case variableScopeOrganization, variableAPIScopeOrg:
		return variableScopeOrganization
	case variableScopeApp:
		return variableScopeApp
	case variableScopeJob, variableAPIScopeAppJob:
		return variableScopeJob
	default:
		return normalized
	}
}

func variableSensitivityFromBool(v types.Bool) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueBool() {
		return variableSensitivitySensitive
	}
	return variableSensitivityNormal
}

func variableBoolFromSensitivity(raw string) types.Bool {
	return types.BoolValue(strings.EqualFold(strings.TrimSpace(raw), variableSensitivitySensitive))
}

func attributeIsSet(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && strings.TrimSpace(v.ValueString()) != ""
}

func configuredVariableValue(plan, config VariableResourceModel) (string, bool) {
	if plan.Sensitive.ValueBool() {
		if attributeIsSet(config.ValueWO) {
			return config.ValueWO.ValueString(), true
		}
		return "", false
	}

	if attributeIsSet(plan.Value) {
		return plan.Value.ValueString(), true
	}

	return "", false
}

func parseVariableImportID(raw string) (scope, orgName, appSlug, jobSlug, variableID string, err error) {
	parts := strings.Split(raw, "/")
	if len(parts) < 2 {
		return "", "", "", "", "", fmt.Errorf("expected import identifier in one of these formats: `global/variable_id`, `organization/org_name/variable_id`, `app/org_name/app_slug/variable_id`, or `job/org_name/app_slug/job_slug/variable_id`")
	}

	switch normalizeVariableScope(parts[0]) {
	case variableScopeGlobal:
		if len(parts) != 2 || parts[1] == "" {
			return "", "", "", "", "", fmt.Errorf("expected import identifier in the format `global/variable_id`")
		}
		return variableScopeGlobal, "", "", "", parts[1], nil
	case variableScopeOrganization:
		if len(parts) != 3 || parts[1] == "" || parts[2] == "" {
			return "", "", "", "", "", fmt.Errorf("expected import identifier in the format `organization/org_name/variable_id`")
		}
		return variableScopeOrganization, parts[1], "", "", parts[2], nil
	case variableScopeApp:
		if len(parts) != 4 || parts[1] == "" || parts[2] == "" || parts[3] == "" {
			return "", "", "", "", "", fmt.Errorf("expected import identifier in the format `app/org_name/app_slug/variable_id`")
		}
		return variableScopeApp, parts[1], parts[2], "", parts[3], nil
	case variableScopeJob:
		if len(parts) != 5 || parts[1] == "" || parts[2] == "" || parts[3] == "" || parts[4] == "" {
			return "", "", "", "", "", fmt.Errorf("expected import identifier in the format `job/org_name/app_slug/job_slug/variable_id`")
		}
		return variableScopeJob, parts[1], parts[2], parts[3], parts[4], nil
	default:
		return "", "", "", "", "", fmt.Errorf("unknown variable scope %q in import identifier", parts[0])
	}
}

func createVariableByScope(ctx context.Context, client *sdk.ClientWithResponses, data VariableResourceModel, body sdk.CreateVariableRequest) (sdk.Variable, error) {
	switch normalizeVariableScope(data.Scope.ValueString()) {
	case variableScopeGlobal:
		rsp, err := client.CreateGlobalVariableWithResponse(ctx, body)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON201 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 201 response when creating global variable, got %s", rsp.Status())
		}
		return rsp.JSON201.Data, nil
	case variableScopeOrganization:
		rsp, err := client.CreateOrganizationVariableWithResponse(ctx, data.OrgName.ValueString(), body)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: data.ID.ValueString()}
		}
		if rsp.JSON201 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 201 response when creating organization variable, got %s", rsp.Status())
		}
		return rsp.JSON201.Data, nil
	case variableScopeApp:
		rsp, err := client.CreateAppVariableWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), body)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: data.ID.ValueString()}
		}
		if rsp.JSON201 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 201 response when creating app variable, got %s", rsp.Status())
		}
		return rsp.JSON201.Data, nil
	default:
		rsp, err := client.CreateAppJobVariableWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), body)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: data.ID.ValueString()}
		}
		if rsp.JSON201 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 201 response when creating job variable, got %s", rsp.Status())
		}
		return rsp.JSON201.Data, nil
	}
}

func getVariableByScope(ctx context.Context, client *sdk.ClientWithResponses, data VariableResourceModel, variableID openapi_types.UUID) (sdk.Variable, error) {
	switch normalizeVariableScope(data.Scope.ValueString()) {
	case variableScopeGlobal:
		rsp, err := client.GetGlobalVariableWithResponse(ctx, variableID)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.JSON200 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 200 response when reading global variable %q, got %s", variableID.String(), rsp.Status())
		}
		return rsp.JSON200.Data, nil
	case variableScopeOrganization:
		rsp, err := client.GetOrganizationVariableWithResponse(ctx, data.OrgName.ValueString(), variableID)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.JSON200 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 200 response when reading organization variable %q, got %s", variableID.String(), rsp.Status())
		}
		return rsp.JSON200.Data, nil
	case variableScopeApp:
		rsp, err := client.GetAppVariableWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), variableID)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.JSON200 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 200 response when reading app variable %q, got %s", variableID.String(), rsp.Status())
		}
		return rsp.JSON200.Data, nil
	default:
		rsp, err := client.GetAppJobVariableWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), variableID)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.JSON200 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 200 response when reading job variable %q, got %s", variableID.String(), rsp.Status())
		}
		return rsp.JSON200.Data, nil
	}
}

func updateVariableByScope(ctx context.Context, client *sdk.ClientWithResponses, data VariableResourceModel, variableID openapi_types.UUID, body sdk.UpdateVariableRequest) (sdk.Variable, error) {
	switch normalizeVariableScope(data.Scope.ValueString()) {
	case variableScopeGlobal:
		rsp, err := client.UpdateGlobalVariableWithResponse(ctx, variableID, body)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.JSON200 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 200 response when updating global variable %q, got %s", variableID.String(), rsp.Status())
		}
		return rsp.JSON200.Data, nil
	case variableScopeOrganization:
		rsp, err := client.UpdateOrganizationVariableWithResponse(ctx, data.OrgName.ValueString(), variableID, body)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.JSON200 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 200 response when updating organization variable %q, got %s", variableID.String(), rsp.Status())
		}
		return rsp.JSON200.Data, nil
	case variableScopeApp:
		rsp, err := client.UpdateAppVariableWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), variableID, body)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.JSON200 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 200 response when updating app variable %q, got %s", variableID.String(), rsp.Status())
		}
		return rsp.JSON200.Data, nil
	default:
		rsp, err := client.UpdateAppJobVariableWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), variableID, body)
		if err != nil {
			return sdk.Variable{}, err
		}
		if rsp.JSON404 != nil {
			return sdk.Variable{}, &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.JSON200 == nil {
			return sdk.Variable{}, fmt.Errorf("expected 200 response when updating job variable %q, got %s", variableID.String(), rsp.Status())
		}
		return rsp.JSON200.Data, nil
	}
}

func deleteVariableByScope(ctx context.Context, client *sdk.ClientWithResponses, data VariableResourceModel, variableID openapi_types.UUID) error {
	switch normalizeVariableScope(data.Scope.ValueString()) {
	case variableScopeGlobal:
		rsp, err := client.DeleteGlobalVariableWithResponse(ctx, variableID)
		if err != nil {
			return err
		}
		if rsp.JSON404 != nil {
			return &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
			return fmt.Errorf("expected 200/204 response when deleting global variable %q, got %s", variableID.String(), rsp.Status())
		}
		return nil
	case variableScopeOrganization:
		rsp, err := client.DeleteOrganizationVariableWithResponse(ctx, data.OrgName.ValueString(), variableID)
		if err != nil {
			return err
		}
		if rsp.JSON404 != nil {
			return &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
			return fmt.Errorf("expected 200/204 response when deleting organization variable %q, got %s", variableID.String(), rsp.Status())
		}
		return nil
	case variableScopeApp:
		rsp, err := client.DeleteAppVariableWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), variableID)
		if err != nil {
			return err
		}
		if rsp.JSON404 != nil {
			return &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
			return fmt.Errorf("expected 200/204 response when deleting app variable %q, got %s", variableID.String(), rsp.Status())
		}
		return nil
	default:
		rsp, err := client.DeleteAppJobVariableWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), variableID)
		if err != nil {
			return err
		}
		if rsp.JSON404 != nil {
			return &variableNotFoundError{scope: data.Scope.ValueString(), id: variableID.String()}
		}
		if rsp.StatusCode() != http.StatusNoContent && rsp.StatusCode() != http.StatusOK {
			return fmt.Errorf("expected 200/204 response when deleting job variable %q, got %s", variableID.String(), rsp.Status())
		}
		return nil
	}
}

func getVariableDefaultValueByScope(ctx context.Context, client *sdk.ClientWithResponses, data VariableResourceModel, variableID openapi_types.UUID) (*sdk.VariableValue, error) {
	switch normalizeVariableScope(data.Scope.ValueString()) {
	case variableScopeGlobal:
		rsp, err := client.GetGlobalVariableDefaultValueWithResponse(ctx, variableID)
		if err != nil {
			return nil, err
		}
		if rsp.JSON404 != nil {
			return nil, nil
		}
		if rsp.JSON200 == nil {
			return nil, fmt.Errorf("expected 200 response when reading global variable default value %q, got %s", variableID.String(), rsp.Status())
		}
		return &rsp.JSON200.Data, nil
	case variableScopeOrganization:
		rsp, err := client.GetOrganizationVariableDefaultValueWithResponse(ctx, data.OrgName.ValueString(), variableID)
		if err != nil {
			return nil, err
		}
		if rsp.JSON404 != nil {
			return nil, nil
		}
		if rsp.JSON200 == nil {
			return nil, fmt.Errorf("expected 200 response when reading organization variable default value %q, got %s", variableID.String(), rsp.Status())
		}
		return &rsp.JSON200.Data, nil
	case variableScopeApp:
		rsp, err := client.GetAppVariableDefaultValueWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), variableID)
		if err != nil {
			return nil, err
		}
		if rsp.JSON404 != nil {
			return nil, nil
		}
		if rsp.JSON200 == nil {
			return nil, fmt.Errorf("expected 200 response when reading app variable default value %q, got %s", variableID.String(), rsp.Status())
		}
		return &rsp.JSON200.Data, nil
	default:
		rsp, err := client.GetAppJobVariableDefaultValueWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), variableID)
		if err != nil {
			return nil, err
		}
		if rsp.JSON404 != nil {
			return nil, nil
		}
		if rsp.JSON200 == nil {
			return nil, fmt.Errorf("expected 200 response when reading job variable default value %q, got %s", variableID.String(), rsp.Status())
		}
		return &rsp.JSON200.Data, nil
	}
}

func upsertVariableDefaultValueByScope(ctx context.Context, client *sdk.ClientWithResponses, data VariableResourceModel, variableID openapi_types.UUID, value string) (*sdk.VariableValue, error) {
	body := sdk.UpsertVariableValueRequest{
		Value: &value,
	}

	switch normalizeVariableScope(data.Scope.ValueString()) {
	case variableScopeGlobal:
		rsp, err := client.UpsertGlobalVariableDefaultValueWithResponse(ctx, variableID, body)
		if err != nil {
			return nil, err
		}
		if rsp.JSON200 == nil {
			return nil, fmt.Errorf("expected 200 response when upserting global variable default value %q, got %s", variableID.String(), rsp.Status())
		}
		return &rsp.JSON200.Data, nil
	case variableScopeOrganization:
		rsp, err := client.UpsertOrganizationVariableDefaultValueWithResponse(ctx, data.OrgName.ValueString(), variableID, body)
		if err != nil {
			return nil, err
		}
		if rsp.JSON200 == nil {
			return nil, fmt.Errorf("expected 200 response when upserting organization variable default value %q, got %s", variableID.String(), rsp.Status())
		}
		return &rsp.JSON200.Data, nil
	case variableScopeApp:
		rsp, err := client.UpsertAppVariableDefaultValueWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), variableID, body)
		if err != nil {
			return nil, err
		}
		if rsp.JSON200 == nil {
			return nil, fmt.Errorf("expected 200 response when upserting app variable default value %q, got %s", variableID.String(), rsp.Status())
		}
		return &rsp.JSON200.Data, nil
	default:
		rsp, err := client.UpsertAppJobVariableDefaultValueWithResponse(ctx, data.OrgName.ValueString(), data.AppSlug.ValueString(), data.JobSlug.ValueString(), variableID, body)
		if err != nil {
			return nil, err
		}
		if rsp.JSON200 == nil {
			return nil, fmt.Errorf("expected 200 response when upserting job variable default value %q, got %s", variableID.String(), rsp.Status())
		}
		return &rsp.JSON200.Data, nil
	}
}

func stateFromVariable(base VariableResourceModel, variableRecord sdk.Variable, defaultRow *sdk.VariableValue) VariableResourceModel {
	description := types.StringNull()
	if variableRecord.Description != nil {
		description = types.StringValue(*variableRecord.Description)
	} else if !base.Description.IsUnknown() {
		description = base.Description
	}

	valueType := types.StringValue(variableRecord.ValueType)
	if variableRecord.ValueType == "" && !base.ValueType.IsUnknown() && !base.ValueType.IsNull() {
		valueType = base.ValueType
	}

	value := types.StringNull()
	if !strings.EqualFold(variableRecord.Sensitivity, variableSensitivitySensitive) {
		switch {
		case defaultRow != nil && defaultRow.Value != nil:
			value = types.StringValue(*defaultRow.Value)
		case variableRecord.DefaultValue != nil:
			value = types.StringValue(*variableRecord.DefaultValue)
		case !base.Value.IsUnknown():
			value = base.Value
		}
	}

	valueID := types.StringNull()
	if defaultRow != nil {
		valueID = types.StringValue(defaultRow.Id.String())
	} else if !base.ValueID.IsUnknown() {
		valueID = base.ValueID
	}

	createdAt := types.StringNull()
	if !variableRecord.CreatedAt.IsZero() {
		createdAt = types.StringValue(variableRecord.CreatedAt.Format(time.RFC3339))
	}

	updatedAt := types.StringNull()
	if !variableRecord.UpdatedAt.IsZero() {
		updatedAt = types.StringValue(variableRecord.UpdatedAt.Format(time.RFC3339))
	}

	return VariableResourceModel{
		ID:          types.StringValue(variableRecord.Id.String()),
		Scope:       types.StringValue(normalizeVariableScope(variableRecord.Scope)),
		OrgName:     base.OrgName,
		AppSlug:     base.AppSlug,
		JobSlug:     base.JobSlug,
		Key:         types.StringValue(variableRecord.Key),
		Description: description,
		Sensitive:   variableBoolFromSensitivity(variableRecord.Sensitivity),
		ValueType:   valueType,
		Value:       value,
		ValueWO:     types.StringNull(),
		ValueID:     valueID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
