package codegen

import (
	"github.com/raviqqe/stg/ast"
	"github.com/raviqqe/stg/codegen/llir"
	"github.com/raviqqe/stg/codegen/names"
	"github.com/raviqqe/stg/types"
	"llvm.org/llvm/bindings/go/llvm"
)

const environmentArgumentName = "environment"

type moduleGenerator struct {
	module          llvm.Module
	globalVariables map[string]llvm.Value
	typeGenerator   typeGenerator
}

func newModuleGenerator(m llvm.Module, ds []ast.ConstructorDefinition) (*moduleGenerator, error) {
	g := newTypeGenerator(m)
	cs := make(map[string]llvm.Value, len(ds))

	for _, d := range ds {
		f := llvm.AddFunction(m, d.Name(), g.GenerateConstructorFunction(d.Type(), d.Index()))

		b := llvm.NewBuilder()
		b.SetInsertPointAtEnd(llvm.AddBasicBlock(f, ""))

		if len(d.Type().Constructors()) == 1 {
			b.CreateAggregateRet(f.Params())
		} else {
			p := b.CreateAlloca(f.Type().ElementType().ReturnType(), "")

			b.CreateStore(
				llvm.ConstInt(llvm.Int32Type(), uint64(d.Index()), false),
				b.CreateStructGEP(p, 0, ""),
			)

			pp := b.CreateBitCast(
				b.CreateStructGEP(p, 1, ""),
				llir.PointerType(g.GenerateConstructorElements(d.Type().Constructors()[d.Index()])),
				"",
			)

			for i, v := range f.Params() {
				b.CreateStore(v, b.CreateStructGEP(pp, i, ""))
			}

			b.CreateRet(b.CreateLoad(p, ""))
		}

		if err := llvm.VerifyFunction(f, llvm.AbortProcessAction); err != nil {
			return nil, err
		}

		cs[d.Name()] = f
	}

	return &moduleGenerator{m, cs, g}, nil
}

func (g *moduleGenerator) Generate(bs []ast.Bind) error {
	for _, b := range bs {
		g.globalVariables[b.Name()] = llvm.AddGlobal(
			g.module,
			g.typeGenerator.GenerateSizedClosure(b.Lambda()),
			b.Name(),
		)
	}

	for _, b := range bs {
		v := g.globalVariables[b.Name()]
		f, err := g.createLambda(b.Name(), b.Lambda())

		if err != nil {
			return err
		} else if err := llvm.VerifyFunction(f, llvm.AbortProcessAction); err != nil {
			return err
		}

		v.SetInitializer(
			llvm.ConstStruct(
				[]llvm.Value{f, llvm.ConstNull(v.Type().ElementType().StructElementTypes()[1])},
				false,
			),
		)
	}

	return llvm.VerifyModule(g.module, llvm.AbortProcessAction)
}

func (g *moduleGenerator) createLambda(n string, l ast.Lambda) (llvm.Value, error) {
	f := llir.AddFunction(
		g.module,
		names.ToEntry(n),
		g.typeGenerator.GenerateLambdaEntryFunction(l),
	)

	b := llvm.NewBuilder()
	b.SetInsertPointAtEnd(llvm.AddBasicBlock(f, ""))

	v, err := newFunctionBodyGenerator(
		b,
		g.createLogicalEnvironment(f, b, l),
		g.createLambda,
	).Generate(l.Body())

	if err != nil {
		return llvm.Value{}, err
	} else if _, ok := l.ResultType().(types.Boxed); ok && l.IsThunk() {
		// TODO: Steal child thunks in a thread-safe way.
		// TODO: Use loop to unbox children recursively.
		v = forceThunk(b, v, g.typeGenerator)
	}

	if l.IsUpdatable() {
		b.CreateStore(v, b.CreateBitCast(f.FirstParam(), llir.PointerType(v.Type()), ""))
		b.CreateStore(
			g.createUpdatedEntryFunction(n, f.Type().ElementType()),
			b.CreateGEP(
				b.CreateBitCast(f.FirstParam(), llir.PointerType(f.Type()), ""),
				[]llvm.Value{llvm.ConstIntFromString(llvm.Int32Type(), "-1", 10)},
				"",
			),
		)
	}

	b.CreateRet(v)

	return f, nil
}

func (g *moduleGenerator) createUpdatedEntryFunction(n string, t llvm.Type) llvm.Value {
	f := llir.AddFunction(g.module, names.ToUpdatedEntry(n), t)
	f.FirstParam().SetName(environmentArgumentName)

	b := llvm.NewBuilder()
	b.SetInsertPointAtEnd(llvm.AddBasicBlock(f, ""))
	b.CreateRet(
		b.CreateLoad(
			b.CreateBitCast(
				f.FirstParam(), llir.PointerType(f.Type().ElementType().ReturnType()),
				""),
			"",
		),
	)

	return f
}

func (g moduleGenerator) createLogicalEnvironment(f llvm.Value, b llvm.Builder, l ast.Lambda) map[string]llvm.Value {
	vs := copyVariables(g.globalVariables)

	e := b.CreateBitCast(
		f.FirstParam(),
		llir.PointerType(g.typeGenerator.GenerateEnvironment(l)),
		"",
	)

	for i, n := range l.FreeVariableNames() {
		vs[n] = b.CreateLoad(b.CreateStructGEP(e, i, ""), "")
	}

	for i, n := range append([]string{environmentArgumentName}, l.ArgumentNames()...) {
		v := f.Param(i)
		v.SetName(n)
		vs[n] = v
	}

	return vs
}
