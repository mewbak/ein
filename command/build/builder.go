package build

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ein-lang/ein/command/compile"
	"github.com/ein-lang/ein/command/parse"
	"llvm.org/llvm/bindings/go/llvm"
)

type builder struct {
	runtimeDirectory, moduleRootDirectory string
	objectCache                           objectCache
}

func newBuilder(runtimeDir, rootDir, cacheDir string) builder {
	return builder{runtimeDir, rootDir, newObjectCache(cacheDir, rootDir)}
}

func (b builder) Build(f string) error {
	o, err := b.BuildModule(f)

	if err != nil {
		return err
	}

	if ok, err := b.isMainModule(f); err != nil {
		return err
	} else if !ok {
		return nil
	}

	bs, err := exec.Command(
		"cc",
		o,
		b.resolveRuntimeLibrary("runtime/target/release/libio.a"),
		b.resolveRuntimeLibrary("runtime/target/release/libcore.a"),
		"-ldl",
		"-lgc",
		"-lpthread",
	).CombinedOutput()

	os.Stderr.Write(bs)

	return err
}

func (b builder) BuildModule(f string) (string, error) {
	if f, ok, err := b.objectCache.Get(f); err != nil {
		return "", err
	} else if ok {
		return f, nil
	}

	bs, err := b.buildModuleWithoutCache(f)

	if err != nil {
		return "", err
	}

	return b.objectCache.Store(f, bs)
}

func (b builder) buildModuleWithoutCache(f string) ([]byte, error) {
	m, err := parse.Parse(f, b.moduleRootDirectory)

	if err != nil {
		return nil, err
	}

	mm, err := compile.Compile(m, nil)

	if err != nil {
		return nil, err
	}

	return b.generateModule(mm)
}

func (b builder) generateModule(m llvm.Module) ([]byte, error) {
	triple := llvm.DefaultTargetTriple()
	target, err := llvm.GetTargetFromTriple(triple)

	if err != nil {
		return nil, err
	}

	buf, err := target.CreateTargetMachine(
		triple,
		"",
		"",
		llvm.CodeGenLevelAggressive,
		llvm.RelocPIC,
		llvm.CodeModelDefault,
	).EmitToMemoryBuffer(m, llvm.ObjectFile)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (b builder) resolveRuntimeLibrary(f string) string {
	return filepath.Join(b.runtimeDirectory, filepath.FromSlash(f))
}

func (b builder) isMainModule(f string) (bool, error) {
	m, err := parse.Parse(f, b.moduleRootDirectory)

	if err != nil {
		return false, err
	}

	return m.IsMainModule(), err
}
