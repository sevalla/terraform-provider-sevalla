package globalenvvar

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/sevalla/terraform-provider-sevalla/internal/client"
)

// GlobalEnvVarResourceModel is the Terraform state model for the sevalla_global_environment_variable resource.
type GlobalEnvVarResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Key         types.String `tfsdk:"key"`
	Value       types.String `tfsdk:"value"`
	IsRuntime   types.Bool   `tfsdk:"is_runtime"`
	IsBuildtime types.Bool   `tfsdk:"is_buildtime"`
	IsSecret    types.Bool   `tfsdk:"is_secret"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// flattenGlobalEnvVar maps a client.GlobalEnvironmentVariable to the Terraform resource model.
func flattenGlobalEnvVar(e *client.GlobalEnvironmentVariable, model *GlobalEnvVarResourceModel) {
	model.ID = types.StringValue(e.ID)
	model.Key = types.StringValue(e.Key)
	// A secret's value is withheld, so keep the one Terraform knows.
	if e.Value != nil {
		model.Value = types.StringValue(*e.Value)
	}
	model.IsSecret = types.BoolValue(e.IsSecret)
	model.IsRuntime = boolOrTrue(e.IsRuntime)
	model.IsBuildtime = boolOrTrue(e.IsBuildtime)
	model.CreatedAt = types.StringValue(e.CreatedAt)
	model.UpdatedAt = types.StringValue(e.UpdatedAt)
}

// buildCreateRequest constructs a CreateGlobalEnvVarRequest from the Terraform plan model.
func buildCreateRequest(model *GlobalEnvVarResourceModel) *client.CreateGlobalEnvVarRequest {
	req := &client.CreateGlobalEnvVarRequest{
		Key:   model.Key.ValueString(),
		Value: model.Value.ValueString(),
	}

	if !model.IsRuntime.IsNull() && !model.IsRuntime.IsUnknown() {
		v := model.IsRuntime.ValueBool()
		req.IsRuntime = &v
	}
	if !model.IsBuildtime.IsNull() && !model.IsBuildtime.IsUnknown() {
		v := model.IsBuildtime.ValueBool()
		req.IsBuildtime = &v
	}
	req.IsSecret = model.IsSecret.ValueBool()

	return req
}

// buildUpdateRequest constructs an UpdateGlobalEnvVarRequest from the Terraform plan model.
func buildUpdateRequest(plan *GlobalEnvVarResourceModel) *client.UpdateGlobalEnvVarRequest {
	req := &client.UpdateGlobalEnvVarRequest{
		Key:   optionalStringPtr(plan.Key),
		Value: optionalStringPtr(plan.Value),
	}

	if !plan.IsRuntime.IsNull() && !plan.IsRuntime.IsUnknown() {
		v := plan.IsRuntime.ValueBool()
		req.IsRuntime = &v
	}
	if !plan.IsBuildtime.IsNull() && !plan.IsBuildtime.IsUnknown() {
		v := plan.IsBuildtime.ValueBool()
		req.IsBuildtime = &v
	}
	// Sending true for a secret is a no-op; false would be rejected.
	req.IsSecret = plan.IsSecret.ValueBool()

	return req
}

// optionalStringPtr returns a *string from a types.String, or nil if null/unknown.
func optionalStringPtr(s types.String) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	v := s.ValueString()
	return &v
}

// boolOrTrue maps a null flag to true, the API's documented default.
func boolOrTrue(b *bool) types.Bool {
	return types.BoolValue(b == nil || *b)
}
