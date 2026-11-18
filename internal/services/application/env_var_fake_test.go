package application_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/sevalla/terraform-provider-sevalla/internal/testutil"
)

// fakeEnvVarAPI serves application env vars and withholds secret values, like the real API.
func fakeEnvVarAPI(t *testing.T) http.HandlerFunc {
	var mu sync.Mutex
	vars := map[string]map[string]any{}
	stored := map[string]any{}
	next := 1

	view := func(id string) map[string]any {
		v := mapWith(vars[id], "value", stored[id])
		if v["is_secret"] == true {
			v["value"] = nil
		}
		return v
	}

	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		id := strings.TrimPrefix(r.URL.Path, "/applications/app-1/env-vars/")
		var in map[string]any
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Errorf("decoding request: %v", err)
			}
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/applications/app-1/env-vars":
			data := []any{}
			for id := range vars {
				data = append(data, view(id))
			}
			json.NewEncoder(w).Encode(map[string]any{"data": data, "total": len(data)})
		case r.Method == http.MethodPost && r.URL.Path == "/applications/app-1/env-vars":
			id = fmt.Sprintf("env-%d", next)
			next++
			vars[id] = map[string]any{
				"id": id, "app_id": "app-1", "key": in["key"], "is_runtime": true, "is_buildtime": true,
				"is_secret": in["is_secret"] == true, "created_at": "2025-01-01T00:00:00.000Z", "updated_at": "2025-01-01T00:00:00.000Z",
			}
			stored[id] = in["value"]
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(view(id))
		case vars[id] != nil && r.Method == http.MethodPut:
			if in["is_secret"] == false && vars[id]["is_secret"] == true {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"message": "A secret cannot be made regular", "status": 400})
				return
			}
			if in["is_secret"] == true {
				vars[id]["is_secret"] = true
			}
			if v, ok := in["value"]; ok {
				stored[id] = v
			}
			json.NewEncoder(w).Encode(view(id))
		case vars[id] != nil && r.Method == http.MethodDelete:
			delete(vars, id)
			json.NewEncoder(w).Encode(map[string]any{"message": "deleted"})
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"message": "Not found", "status": 404})
		}
	}
}

func envVarConfig(value string, secret bool) string {
	return fmt.Sprintf(`
resource "sevalla_application_environment_variable" "test" {
  application_id = "app-1"
  key            = "TOKEN"
  value          = %q
  is_secret      = %t
}
`, value, secret)
}

func TestApplicationEnvVar_FakeAPISecret(t *testing.T) {
	testutil.FakeAPI(t, fakeEnvVarAPI(t))
	const name = "sevalla_application_environment_variable.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: envVarConfig("one", false)},
			{
				// Turning a regular variable secret is an in-place update, and the withheld value causes no drift.
				Config: envVarConfig("one", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "is_secret", "true"),
					resource.TestCheckResourceAttr(name, "value", "one"),
				),
			},
			{
				Config: envVarConfig("two", true),
				Check:  resource.TestCheckResourceAttr(name, "value", "two"),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateId:           "app-1/env-1",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
			{
				// A secret cannot be made regular again, so it is replaced.
				Config: envVarConfig("two", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(name, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: resource.TestCheckResourceAttr(name, "is_secret", "false"),
			},
		},
	})
}
