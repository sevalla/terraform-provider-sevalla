package globalenvvar_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/sevalla/terraform-provider-sevalla/internal/testutil"
)

func TestAccGlobalEnvironmentVariable_basic(t *testing.T) {
	rSuffix := acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
	envKey := fmt.Sprintf("TF_ACC_TEST_%s", rSuffix)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutil.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccGlobalEnvVarConfig_basic(envKey, "initial-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("sevalla_global_environment_variable.test", "id"),
					resource.TestCheckResourceAttr("sevalla_global_environment_variable.test", "key", envKey),
					resource.TestCheckResourceAttr("sevalla_global_environment_variable.test", "value", "initial-value"),
				),
			},
			// Import
			{
				ResourceName:      "sevalla_global_environment_variable.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update value
			{
				Config: testAccGlobalEnvVarConfig_basic(envKey, "updated-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sevalla_global_environment_variable.test", "key", envKey),
					resource.TestCheckResourceAttr("sevalla_global_environment_variable.test", "value", "updated-value"),
				),
			},
			// Make it secret: the API withholds the value from now on, which must not cause drift
			{
				Config: testAccGlobalEnvVarConfig_secret(envKey, "secret-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("sevalla_global_environment_variable.test", "is_secret", "true"),
					resource.TestCheckResourceAttr("sevalla_global_environment_variable.test", "value", "secret-value"),
				),
			},
			// Making it regular again replaces it
			{
				Config: testAccGlobalEnvVarConfig_basic(envKey, "plain-value"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("sevalla_global_environment_variable.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("sevalla_global_environment_variable.test", "is_secret", "false"),
			},
		},
	})
}

func testAccGlobalEnvVarConfig_basic(key, value string) string {
	return fmt.Sprintf(`
resource "sevalla_global_environment_variable" "test" {
  key   = %[1]q
  value = %[2]q
}
`, key, value)
}

func testAccGlobalEnvVarConfig_secret(key, value string) string {
	return fmt.Sprintf(`
resource "sevalla_global_environment_variable" "test" {
  key       = %[1]q
  value     = %[2]q
  is_secret = true
}
`, key, value)
}
