package build_context

import (
	"bytes"
	"github.com/docker/docker/pkg/archive"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// New creates a fake build context
func New(dir string, modifiers ...func(*BuildContext) error) (*BuildContext, error) {
	buildContext := &BuildContext{Dir: dir}
	if dir == "" {
		if err := newDir(buildContext); err != nil {
			return nil, err
		}
	}

	for _, modifier := range modifiers {
		if err := modifier(buildContext); err != nil {
			return nil, err
		}
	}

	return buildContext, nil
}

func newDir(fake *BuildContext) error {
	tmp, err := ioutil.TempDir("", "fake-context")
	if err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0755); err != nil {
		return err
	}
	fake.Dir = tmp
	return nil
}

// WithFile adds the specified file (with content) in the build context
func WithFile(name, content string) func(*BuildContext) error {
	return func(ctx *BuildContext) error {
		return ctx.Add(name, content)
	}
}

// WithDockerfile adds the specified content as Dockerfile in the build context
func WithDockerfile(content string) func(*BuildContext) error {
	return WithFile("Dockerfile", content)
}

// WithFiles adds the specified files in the build context, content is a string
func WithFiles(files map[string]string) func(*BuildContext) error {
	return func(buildContext *BuildContext) error {
		for file, content := range files {
			if err := buildContext.Add(file, content); err != nil {
				return err
			}
		}
		return nil
	}
}

// WithBinaryFiles adds the specified files in the build context, content is binary
func WithBinaryFiles(files map[string]*bytes.Buffer) func(*BuildContext) error {
	return func(buildContext *BuildContext) error {
		for file, content := range files {
			if err := buildContext.Add(file, content.String()); err != nil {
				return err
			}
		}
		return nil
	}
}

// BuildContext creates directories that can be used as a build context
type BuildContext struct {
	Dir string
}

// Add a file at a path, creating directories where necessary
func (f *BuildContext) Add(file, content string) error {
	return f.addFile(file, []byte(content))
}

func (f *BuildContext) addFile(file string, content []byte) error {
	fp := filepath.Join(f.Dir, filepath.FromSlash(file))
	dirpath := filepath.Dir(fp)
	if dirpath != "." {
		if err := os.MkdirAll(dirpath, 0755); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(fp, content, 0644)

}

// Delete a file at a path
func (f *BuildContext) Delete(file string) error {
	fp := filepath.Join(f.Dir, filepath.FromSlash(file))
	return os.RemoveAll(fp)
}

// Close deletes the context
func (f *BuildContext) Close() error {
	return os.RemoveAll(f.Dir)
}

// AsTarReader returns a ReadCloser with the contents of Dir as a tar archive.
func (f *BuildContext) AsTarReader() (io.ReadCloser, error) {
	reader, err := archive.TarWithOptions(f.Dir, &archive.TarOptions{})
	if err != nil {
		return nil, err
	}
	return reader, nil
}
