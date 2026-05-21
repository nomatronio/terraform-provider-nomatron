package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/nomatronio/nomatron/pkg/api/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func handleDestroyPlanResponseDiagnostics(resp *action.InvokeResponse, badRequest *sdk.ApiResponseError, unauthorized *sdk.ApiResponseError, forbidden *sdk.ApiResponseError, notFound *sdk.ApiResponseError, conflict *sdk.ApiResponseError, internal *sdk.ApiResponseError, status string) bool {
	if badRequest != nil {
		resp.Diagnostics.AddError("Failed To Create Destroy Plan", formatAPIError(badRequest))
		return false
	}
	if unauthorized != nil {
		resp.Diagnostics.AddError("Unauthorized", formatAPIError(unauthorized))
		return false
	}
	if forbidden != nil {
		resp.Diagnostics.AddError("Forbidden", formatAPIError(forbidden))
		return false
	}
	if notFound != nil {
		resp.Diagnostics.AddError("Target Not Found", formatAPIError(notFound))
		return false
	}
	if conflict != nil {
		resp.Diagnostics.AddError("Destroy Plan Conflict", formatAPIError(conflict))
		return false
	}
	if internal != nil {
		resp.Diagnostics.AddError("Failed To Create Destroy Plan", formatAPIError(internal))
		return false
	}
	return true
}

func reportDestroyPlanProgress(resp *action.InvokeResponse, prefix string, plan sdk.DestroyPlanResult) {
	if resp.SendProgress == nil {
		return
	}
	operation := "app-level batch"
	if plan.OperationId != nil {
		operation = plan.OperationId.String()
	} else if plan.BatchId != nil {
		operation = plan.BatchId.String()
	}
	approval := "per target"
	if plan.ApprovalState != nil {
		approval = string(*plan.ApprovalState)
	}
	resp.SendProgress(action.InvokeProgressEvent{
		Message: fmt.Sprintf("%s %q created for %d target(s); approval=%s ready_for_apply=%t.", prefix, operation, plan.TargetCount, approval, plan.ReadyForApply),
	})
}

func applyDestroyPlan(ctx context.Context, client *sdk.ClientWithResponses, orgName string, plan sdk.DestroyPlanResult, resp *action.InvokeResponse) {
	planIDs := destroyPlanOperationIDs(plan)
	if len(planIDs) == 0 {
		resp.Diagnostics.AddError(
			"Destroy Plan Not Applyable",
			"The destroy plan response did not include any plan operation IDs to apply.",
		)
		return
	}

	for _, planID := range planIDs {
		rsp, err := client.CreateDestroyApplyWithResponse(ctx, orgName, planID)
		if err != nil {
			resp.Diagnostics.AddError("Failed To Apply Destroy Plan", err.Error())
			return
		}
		if !handleDestroyApplyResponseDiagnostics(&resp.Diagnostics, rsp.JSON400, rsp.JSON401, rsp.JSON403, rsp.JSON404, rsp.JSON409, rsp.JSON500, rsp.Status()) {
			return
		}
		if rsp.JSON202 == nil {
			resp.Diagnostics.AddError(
				"Unexpected API Response",
				fmt.Sprintf("Expected 202 response when applying destroy plan %q in org %q, got %s.", planID.String(), orgName, rsp.Status()),
			)
			return
		}
		if resp.SendProgress != nil {
			resp.SendProgress(action.InvokeProgressEvent{
				Message: fmt.Sprintf("Queued destroy apply operation %q for plan %q with state %q.", rsp.JSON202.Data.OperationId.String(), rsp.JSON202.Data.PlanOperationId.String(), rsp.JSON202.Data.State),
			})
		}
	}
}

func destroyPlanOperationIDs(plan sdk.DestroyPlanResult) []openapi_types.UUID {
	ids := make([]openapi_types.UUID, 0, 1)
	if plan.OperationId != nil {
		ids = append(ids, *plan.OperationId)
	}
	if plan.Items == nil {
		return ids
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		seen[id.String()] = struct{}{}
	}
	for _, item := range *plan.Items {
		if item.PlanOperationId == nil {
			continue
		}
		if _, ok := seen[item.PlanOperationId.String()]; ok {
			continue
		}
		seen[item.PlanOperationId.String()] = struct{}{}
		ids = append(ids, *item.PlanOperationId)
	}
	return ids
}

func handleDestroyApplyResponseDiagnostics(diags *diag.Diagnostics, badRequest *sdk.ApiResponseError, unauthorized *sdk.ApiResponseError, forbidden *sdk.ApiResponseError, notFound *sdk.ApiResponseError, conflict *sdk.ApiResponseError, internal *sdk.ApiResponseError, status string) bool {
	if badRequest != nil {
		diags.AddError("Failed To Apply Destroy Plan", formatAPIError(badRequest))
		return false
	}
	if unauthorized != nil {
		diags.AddError("Unauthorized", formatAPIError(unauthorized))
		return false
	}
	if forbidden != nil {
		diags.AddError("Forbidden", formatAPIError(forbidden))
		return false
	}
	if notFound != nil {
		diags.AddError("Destroy Plan Not Found", formatAPIError(notFound))
		return false
	}
	if conflict != nil {
		diags.AddError("Destroy Apply Conflict", formatAPIError(conflict))
		return false
	}
	if internal != nil {
		diags.AddError("Failed To Apply Destroy Plan", formatAPIError(internal))
		return false
	}
	return true
}
