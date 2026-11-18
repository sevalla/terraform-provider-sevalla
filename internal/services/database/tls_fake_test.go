package database_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/sevalla/terraform-provider-sevalla/internal/testutil"
)

// fakeDatabaseAPI serves one database that starts without TLS. Enabling it
// completes at once, so the provider's ready waits return on the first poll.
func fakeDatabaseAPI(t *testing.T, enableCalls *int) http.HandlerFunc {
	var mu sync.Mutex
	var db map[string]any

	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/databases":
			db = map[string]any{
				"id": "db-1", "name": "db-1", "display_name": "test", "type": "postgresql", "version": "16",
				"status": "ready", "is_suspended": false, "cluster_id": "c-1", "resource_type_id": "rt-1",
				"db_name": "app", "db_user": "app", "tls_enabled": false, "tls_ca_certificate": nil,
				"created_at": "2025-01-01T00:00:00.000Z", "updated_at": "2025-01-01T00:00:00.000Z",
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": "db-1"})
		case db != nil && r.Method == http.MethodGet && r.URL.Path == "/databases/db-1":
			json.NewEncoder(w).Encode(db)
		case db != nil && r.Method == http.MethodPost && r.URL.Path == "/databases/db-1/tls/enable":
			*enableCalls++
			db["tls_enabled"] = true
			db["tls_ca_certificate"] = "-----BEGIN CERTIFICATE-----\nCA\n-----END CERTIFICATE-----\n"
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{"message": "TLS enabling started"})
		case db != nil && r.Method == http.MethodDelete && r.URL.Path == "/databases/db-1":
			db = nil
			json.NewEncoder(w).Encode(map[string]any{"message": "deleted"})
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"message": "Not found", "status": 404})
		}
	}
}

func databaseConfig(tls string) string {
	return `
resource "sevalla_database" "test" {
  display_name     = "test"
  type             = "postgresql"
  version          = "16"
  cluster_id       = "c-1"
  resource_type_id = "rt-1"
  db_name          = "app"
  db_password      = "secret-password"
` + tls + `
}
`
}

func TestDatabaseTLS_FakeAPIEnable(t *testing.T) {
	var enableCalls int
	testutil.FakeAPI(t, fakeDatabaseAPI(t, &enableCalls))
	const name = "sevalla_database.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: databaseConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "tls_enabled", "false"),
					resource.TestCheckNoResourceAttr(name, "tls_ca_certificate"),
				),
			},
			{
				Config: databaseConfig(`  tls_enabled      = true`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "tls_enabled", "true"),
					resource.TestCheckResourceAttrSet(name, "tls_ca_certificate"),
					func(*terraform.State) error {
						if enableCalls != 1 {
							return fmt.Errorf("expected 1 enable call, got %d", enableCalls)
						}
						return nil
					},
				),
			},
			{
				Config:      databaseConfig(`  tls_enabled      = false`),
				ExpectError: regexp.MustCompile(`Value must be "true"`),
			},
			// Back to a valid configuration so the post-test destroy can run.
			{Config: databaseConfig(`  tls_enabled      = true`)},
		},
	})
}
