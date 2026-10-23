package dockerx

import (
	"github.com/containerd/errdefs"
)

func isNotFound(err error) bool { return err != nil && errdefs.IsNotFound(err) }

func isConflict(err error) bool { return err != nil && errdefs.IsConflict(err) }
