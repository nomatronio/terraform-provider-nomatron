package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func lowerCaseString() planmodifier.String {
	return lowerCaseStringModifier{}
}

type lowerCaseStringModifier struct{}

func (m lowerCaseStringModifier) Description(ctx context.Context) string {
	return "Normalizes a configured string to lowercase."
}

func (m lowerCaseStringModifier) MarkdownDescription(ctx context.Context) string {
	return "Normalizes a configured string to lowercase."
}

func (m lowerCaseStringModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	normalized := normalizeProviderString(req.PlanValue.ValueString())
	if normalized == req.PlanValue.ValueString() {
		return
	}

	resp.PlanValue = types.StringValue(normalized)
}

func normalizeProviderString(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
