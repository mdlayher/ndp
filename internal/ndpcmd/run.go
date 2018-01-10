// Package ndpcmd provides the commands for the ndp utility.
package ndpcmd

import (
	"context"
	"fmt"
	"net"

	"github.com/mdlayher/ndp"
)

// Run runs the ndp utility.
func Run(ctx context.Context, c *ndp.Conn, ifi *net.Interface, op string) error {
	switch op {
	case "rs":
		return sendRS(ctx, c, ifi.HardwareAddr)
	default:
		return fmt.Errorf("unrecognized operation: %q", op)
	}
}
