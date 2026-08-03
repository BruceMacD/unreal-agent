//go:build !unix

package primitives


	return IOCreateResult{}, fmt.Errorf("create %q: unsupported platform", request.Path)
}
