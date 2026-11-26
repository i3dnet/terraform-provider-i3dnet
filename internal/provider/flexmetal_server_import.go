package provider

import (
	"context"
	"fmt"
)

// errServerManagedByProduct returns the reason a server cannot be imported when
// another i3D product (e.g. FlexVM) manages it, or nil when it can be imported.
//
// Lookup failures are not reported here: the Read that follows the import
// surfaces them with its usual handling.
func errServerManagedByProduct(ctx context.Context, get getServerFunc, id string) error {
	resp, err := getServerWithRetry(ctx, get, id, readRetryAttempts, readRetryDelay)
	if err != nil || resp == nil || resp.Server == nil {
		return nil
	}
	if resp.Server.ManagedBy == "" {
		return nil
	}

	return fmt.Errorf("server %s is managed by %q and cannot be changed by Terraform as an "+
		"i3dnet_flexmetal_server. Manage it through %s instead (for FlexVM, the i3dnet_flexvm_* resources)",
		id, resp.Server.ManagedBy, resp.Server.ManagedBy)
}
