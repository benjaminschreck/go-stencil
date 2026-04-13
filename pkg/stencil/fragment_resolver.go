package stencil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FragmentResolver resolves fragments on demand.
type FragmentResolver interface {
	ResolveFragment(name string) ([]byte, error)
}

// FragmentResolverFunc adapts a function into a FragmentResolver.
type FragmentResolverFunc func(name string) ([]byte, error)

// ResolveFragment resolves a fragment by name.
func (f FragmentResolverFunc) ResolveFragment(name string) ([]byte, error) {
	return f(name)
}

// SetFragmentResolver configures an optional on-demand fragment resolver.
func (pt *PreparedTemplate) SetFragmentResolver(resolver FragmentResolver) {
	if pt == nil || pt.template == nil {
		return
	}
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.template.mu.Lock()
	defer pt.template.mu.Unlock()
	pt.template.fragmentResolver = resolver
	for name, frag := range pt.template.fragments {
		if frag != nil && frag.resolvedByResolver {
			delete(pt.template.fragments, name)
		}
	}
	if pt.template.resolverMisses == nil {
		pt.template.resolverMisses = make(map[string]bool)
	}
	clear(pt.template.resolverMisses)
	pt.template.invalidateFragmentCachesLocked()
}

// DirectoryFragmentResolver resolves DOCX fragments from a directory.
func DirectoryFragmentResolver(dir string) FragmentResolver {
	return FragmentResolverFunc(func(name string) ([]byte, error) {
		if !isSafeFragmentName(name) {
			return nil, fmt.Errorf("invalid fragment name: %s", name)
		}

		rootAbs, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("resolve fragment root %s: %w", dir, err)
		}
		rootResolved, err := filepath.EvalSymlinks(rootAbs)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("resolve fragment root %s: %w", dir, err)
		}

		path := filepath.Join(rootResolved, name+".docx")
		pathAbs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve fragment %s: %w", name, err)
		}
		if pathAbs != rootResolved && !strings.HasPrefix(pathAbs, rootResolved+string(filepath.Separator)) {
			return nil, fmt.Errorf("fragment path escapes resolver root: %s", name)
		}

		if _, err := os.Lstat(pathAbs); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("stat fragment %s: %w", name, err)
		}

		resolvedPath, err := filepath.EvalSymlinks(pathAbs)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("resolve fragment %s: %w", name, err)
		}
		if resolvedPath != rootResolved && !strings.HasPrefix(resolvedPath, rootResolved+string(filepath.Separator)) {
			return nil, fmt.Errorf("fragment path escapes resolver root: %s", name)
		}

		content, err := os.ReadFile(resolvedPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("read fragment %s: %w", name, err)
		}
		return content, nil
	})
}

func isSafeFragmentName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if filepath.IsAbs(name) {
		return false
	}
	if filepath.Clean(name) != name {
		return false
	}
	return !strings.ContainsAny(name, `/\`)
}
