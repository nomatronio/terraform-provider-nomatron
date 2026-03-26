package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
)

var _ datasource.DataSource = &UsersDataSource{}

type UsersDataSource struct {
	client *sdk.ClientWithResponses
}

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

type UsersDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	User     types.Object `tfsdk:"user"`
	Users    types.List   `tfsdk:"users"`
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetch a single user by username, or all users when username is omitted.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Stable Terraform state ID.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, fetch a single user by username. If omitted, fetch all users.",
			},
			"user": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Single user result when username is provided.",
				Attributes:          userSchemaAttributes(),
			},
			"users": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of users when username is omitted.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: userSchemaAttributes(),
				},
			},
		},
	}
}

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError(
			"Client Not Configured",
			"The provider client was not configured for the users data source.",
		)
		return
	}

	userObjectType := types.ObjectType{AttrTypes: userAttrTypes()}

	if !data.Username.IsNull() && !data.Username.IsUnknown() && data.Username.ValueString() != "" {
		username := data.Username.ValueString()

		rsp, err := d.client.GetUserWithResponse(ctx, username)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed To Read User",
				err.Error(),
			)
			return
		}

		if rsp.JSON404 != nil {
			resp.Diagnostics.AddError(
				"User Not Found",
				fmt.Sprintf("No user was found with username %q.", username),
			)
			return
		}

		if rsp.JSON200 == nil {
			resp.Diagnostics.AddError(
				"Unexpected API Response",
				fmt.Sprintf("Expected 200 response when reading user %q, got %s.", username, rsp.Status()),
			)
			return
		}

		userValue, diags := flattenUser(rsp.JSON200.Data)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.ID = types.StringValue("user:" + username)
		data.User = userValue
		data.Users = types.ListNull(userObjectType)
	} else {
		rsp, err := d.client.ListUsersWithResponse(ctx, nil)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed To List Users",
				err.Error(),
			)
			return
		}

		if rsp.JSON200 == nil {
			resp.Diagnostics.AddError(
				"Unexpected API Response",
				fmt.Sprintf("Expected 200 response when listing users, got %s.", rsp.Status()),
			)
			return
		}

		values := make([]attr.Value, 0, len(rsp.JSON200.Data))
		for _, u := range rsp.JSON200.Data {
			userValue, diags := flattenUser(u)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			values = append(values, userValue)
		}

		usersValue, diags := types.ListValue(userObjectType, values)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.ID = types.StringValue("users")
		data.User = types.ObjectNull(userAttrTypes())
		data.Users = usersValue
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sdk.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sdk.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func userSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The user ID.",
		},
		"name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Display name for the user.",
		},
		"username": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The username for the user.",
		},
		"is_active": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the user is active.",
		},
		"auth_provider": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Authentication provider for the user.",
		},
		"created_at": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Timestamp when the user was created.",
		},
		"created_by": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Actor that created the user.",
		},
		"metadata": schema.MapAttribute{
			ElementType:         types.StringType,
			Computed:            true,
			MarkdownDescription: "Metadata attached to the user.",
		},
	}
}

func userAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":            types.StringType,
		"name":          types.StringType,
		"username":      types.StringType,
		"is_active":     types.BoolType,
		"auth_provider": types.StringType,
		"created_at":    types.StringType,
		"created_by":    types.StringType,
		"metadata":      types.MapType{ElemType: types.StringType},
	}
}

func flattenUser(u sdk.User) (types.Object, diag.Diagnostics) {
	metadata := types.MapNull(types.StringType)
	if u.Metadata != nil {
		mv, diags := types.MapValueFrom(context.Background(), types.StringType, *u.Metadata)
		if diags.HasError() {
			return types.ObjectNull(userAttrTypes()), diags
		}
		metadata = mv
	}

	createdBy := flattenCreatedBy(u.CreatedBy)

	return types.ObjectValue(
		userAttrTypes(),
		map[string]attr.Value{
			"id":            types.StringValue(u.Id.String()),
			"name":          types.StringValue(u.Name),
			"username":      types.StringValue(u.Username),
			"is_active":     types.BoolValue(u.IsActive),
			"auth_provider": types.StringValue(u.AuthProvider),
			"created_at":    types.StringValue(u.CreatedAt.Format(time.RFC3339)),
			"created_by":    types.StringValue(createdBy),
			"metadata":      metadata,
		},
	)
}

func flattenCreatedBy(v sdk.User_CreatedBy) string {
	if asUUID, err := v.AsUserCreatedBy0(); err == nil {
		return fmt.Sprint(asUUID)
	}
	if asString, err := v.AsUserCreatedBy1(); err == nil {
		return string(asString)
	}
	return ""
}
