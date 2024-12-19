package mock_http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"terraform-provider-i3d/internal/provider/resource_servers"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// callFlexMetalAPI is a placeholder for the API call logic to the FlexMetal API.
func CallFlexMetalAPI(server *resource_servers.ServersModel, method string, path string, body []byte) diag.Diagnostics {
	var diags diag.Diagnostics

	// Create an HTTP client
	// client := &http.Client{Timeout: 10 * time.Second}

	// Build the full URL using the server's endpoint and the provided path
	url := fmt.Sprintf("%s/%s", "", path)

	// Create the request
	var req *http.Request
	var err error
	if len(body) > 0 {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		diags.AddError("API Request Error", fmt.Sprintf("Failed to create HTTP request: %s", err))
		return diags
	}

	// Set the required headers (API Key and Content-Type)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PRIVATE-TOKEN", os.Getenv("FLEXMETAL_API_KEY"))

	// Execute the request
	// resp, err := client.Do(req)
	// if err != nil {
	// 	diags.AddError("API Request Error", fmt.Sprintf("Failed to execute API request: %s", err))
	// 	return diags
	// }
	// defer resp.Body.Close()

	// // Read the response body
	// responseBody, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	diags.AddError("API Response Error", fmt.Sprintf("Failed to read response body: %s", err))
	// 	return diags
	// }

	// Check for non-2xx response codes
	// if resp.StatusCode < 200 || resp.StatusCode >= 300 {
	// 	diags.AddError("API Response Error", fmt.Sprintf("API responded with non-2xx status: %d, reason: %s, sent body: %s", resp.StatusCode, responseBody, body))
	// 	return diags
	// }
	responseBody := []byte(`
	[ {
				"uuid": "0192c3b6-611f-7033-810a-2fe613e774d4",
				"name": "bm7-std8-cp-2",
				"status": "delivered",
				"statusMessage": "OK",
				"location": {
					"id": 177,
					"name": "EU: Montreuil 1"
				},
				"instanceType": {
					"id": 129,
					"name": "bm7.std.8"
				},
				"os": {
					"slug": "talos-omni-177"
				},
				"ipAddresses": [
					{
						"ipAddress": "89.104.168.49"
					}
				],
				"createdAt": 1729860362,
				"deliveredAt": null,
				"releasedAt": null
			}
		] 
	`)

	// Optionally log or parse the response
	fmt.Printf("API Response: %s\n", responseBody)

	// parse the response body
	unmarshalledData := []map[string]any{}

	err = json.Unmarshal(responseBody, &unmarshalledData)
	if err != nil {
		diags.AddError("API Response Error", fmt.Sprintf("Failed to parse response body: %s, body content: %s", err, responseBody))
		return diags
	}

	for _, answer := range unmarshalledData {
		server.Uuid = basetypes.NewStringValue(answer["uuid"].(string))
		server.Status = basetypes.NewStringValue(answer["status"].(string))
		server.StatusMessage = basetypes.NewStringValue(answer["statusMessage"].(string))
		server.PostInstallScript = basetypes.NewStringValue("")
		if answer["ipAddresses"] != nil {
			for _, ip := range answer["ipAddresses"].([]interface{}) {

				ipAddress := basetypes.NewObjectValueMust(
					map[string]attr.Type{
						"ip_address": basetypes.StringType{},
					},
					map[string]attr.Value{
						"ip_address": basetypes.NewStringValue(ip.(map[string]any)["ipAddress"].(string)),
					})
				values := append(server.IpAddresses.Elements(), ipAddress)
				server.IpAddresses = types.ListValueMust(
					basetypes.ObjectType{
						AttrTypes: map[string]attr.Type{
							"ip_address": basetypes.StringType{},
						},
					},
					values,
				)

			}
		} else {
			server.IpAddresses = basetypes.NewListUnknown(resource_servers.IpAddressesType{})
		}
		if answer["createdAt"] != nil {
			server.CreatedAt = basetypes.NewInt64Value(int64(answer["createdAt"].(float64)))
		} else {
			server.CreatedAt = basetypes.NewInt64Value(0)
		}
		if answer["deliveredAt"] != nil {
			server.DeliveredAt = basetypes.NewInt64Value(int64(answer["deliveredAt"].(float64)))
		} else {
			server.DeliveredAt = basetypes.NewInt64Value(0)
		}
		if answer["releasedAt"] != nil {
			server.ReleasedAt = basetypes.NewInt64Value(int64(answer["releasedAt"].(float64)))
		} else {
			server.ReleasedAt = basetypes.NewInt64Value(0)
		}

	}

	return diags
}
