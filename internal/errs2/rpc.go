// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package errs2

import (
	"strings"

	"github.com/zeebo/errs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsRPC checks if err contains an RPC error with the given status code.
func IsRPC(err error, code codes.Code) bool {
	if err != nil && strings.Contains(err.Error(), "code = "+code.String()) {
		return true
	}
	return errs.IsFunc(err, func(err error) bool {
		return status.Code(err) == code
	})
}
