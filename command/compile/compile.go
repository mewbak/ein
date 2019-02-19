package compile

import (
	"path"

	"github.com/ein-lang/ein/command/ast"
	"github.com/ein-lang/ein/command/compile/desugar"
	"github.com/ein-lang/ein/command/compile/tinfer"
	coreast "github.com/ein-lang/ein/command/core/ast"
	corecompile "github.com/ein-lang/ein/command/core/compile"
	"llvm.org/llvm/bindings/go/llvm"
)

// Compile compiles a module into a module in the core language with imported modules.
func Compile(m ast.Module, ms []ast.Module) (llvm.Module, error) {
	mm, err := compileToCore(m, ms)

	if err != nil {
		return llvm.Module{}, err
	}

	return corecompile.Compile(renameGlobalVariables(mm, m, ms))
}

func compileToCore(m ast.Module, ms []ast.Module) (coreast.Module, error) {
	m, err := tinfer.InferTypes(desugar.WithoutTypes(m), ms)

	if err != nil {
		return coreast.Module{}, err
	}

	return newCompiler().Compile(desugar.WithTypes(m), ms)
}

func renameGlobalVariables(m coreast.Module, mm ast.Module, ms []ast.Module) coreast.Module {
	vs := make(map[string]string, len(m.Binds()))

	for _, m := range ms {
		for _, b := range m.ExportedBinds() {
			vs[path.Base(string(m.Name()))+"."+b.Name()] = string(m.Name()) + "." + b.Name()
		}
	}

	ds := make([]coreast.Declaration, 0, len(m.Declarations()))

	for _, d := range m.Declarations() {
		ds = append(ds, coreast.NewDeclaration(vs[d.Name()], d.Lambda()))
	}

	bs := make([]coreast.Bind, 0, len(m.Binds()))

	for _, b := range m.Binds() {
		s := string(mm.Name()) + "." + b.Name()

		vs[b.Name()] = s

		if b.Name() == ast.MainFunctionName {
			bs = append(bs, coreast.NewBind("ein_main", b.Lambda()))
		} else {
			bs = append(bs, coreast.NewBind(s, b.Lambda()))
		}
	}

	return coreast.NewModule(ds, bs).RenameVariables(vs)
}
