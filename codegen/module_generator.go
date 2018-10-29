package codegen

import (
	"github.com/raviqqe/stg/ast"
	"github.com/raviqqe/stg/types"
	"llvm.org/llvm/bindings/go/llvm"
)

type moduleGenerator struct {
	module          llvm.Module
	globalVariables map[string]llvm.Value
}

func newModuleGenerator(m llvm.Module) *moduleGenerator {
	return &moduleGenerator{m, map[string]llvm.Value{}}
}

func (g *moduleGenerator) Generate(bs []ast.Bind) error {
	for _, b := range bs {
		f, err := g.createLambda(b.Name(), b.Lambda())

		if err != nil {
			return err
		} else if err := llvm.VerifyFunction(f, llvm.AbortProcessAction); err != nil {
			return err
		}

		g.createClosure(b.Name(), f)
	}

	return llvm.VerifyModule(g.module, llvm.AbortProcessAction)
}

func (g *moduleGenerator) createLambda(n string, l ast.Lambda) (llvm.Value, error) {
	t := types.Unbox(l.ResultType()).LLVMType()

	f := llvm.AddFunction(
		g.module,
		toEntryName(n),
		llvm.FunctionType(
			t,
			append(
				[]llvm.Type{types.NewEnvironment(0).LLVMPointerType()},
				types.ToLLVMTypes(l.ArgumentTypes())...,
			),
			false,
		),
	)

	b := llvm.NewBuilder()
	v, err := newFunctionBodyGenerator(f, b, l.ArgumentNames(), g.globalVariables).Generate(l.Body())

	if err != nil {
		return llvm.Value{}, err
	}

	p := g.environmentToEntryFunctionPointer(b, f.FirstParam(), t)

	if _, ok := l.ResultType().(types.Boxed); ok {
		v = g.unboxResultType(b, v, p)
	}

	if l.IsUpdatable() {
		b.CreateStore(v, b.CreateBitCast(f.FirstParam(), llvm.PointerType(v.Type(), 0), ""))
		b.CreateStore(g.createUpdatedEntryFunction(n, t), p)
	}

	b.CreateRet(v)

	return f, nil
}

func (g *moduleGenerator) unboxResultType(b llvm.Builder, v, p llvm.Value) llvm.Value {
	return b.CreateCall(
		b.CreateLoad(b.CreateStructGEP(v, 0, ""), ""),
		[]llvm.Value{
			b.CreateBitCast(
				b.CreateStructGEP(v, 1, ""),
				types.NewEnvironment(0).LLVMPointerType(),
				"",
			),
		},
		"",
	)
}

func (g *moduleGenerator) createClosure(n string, f llvm.Value) {
	e := types.NewEnvironment(g.getTypeSize(f.Type().ElementType().ReturnType())).LLVMType()

	v := llvm.AddGlobal(
		g.module,
		llvm.StructType([]llvm.Type{f.Type(), e}, false),
		n,
	)
	v.SetInitializer(llvm.ConstStruct([]llvm.Value{f, llvm.ConstNull(e)}, false))

	g.globalVariables[n] = v
}

func (g *moduleGenerator) createUpdatedEntryFunction(n string, t llvm.Type) llvm.Value {
	f := llvm.AddFunction(
		g.module,
		toUpdatedEntryName(n),
		llvm.FunctionType(t, []llvm.Type{types.NewEnvironment(0).LLVMPointerType()}, false),
	)

	b := llvm.NewBuilder()
	b.SetInsertPointAtEnd(llvm.AddBasicBlock(f, ""))
	b.CreateRet(b.CreateLoad(b.CreateBitCast(f.FirstParam(), llvm.PointerType(t, 0), ""), ""))

	return f
}

func (g *moduleGenerator) getTypeSize(t llvm.Type) int {
	return int(llvm.NewTargetData(g.module.DataLayout()).TypeAllocSize(t))
}

func (g *moduleGenerator) environmentToEntryFunctionPointer(
	b llvm.Builder, v llvm.Value, t llvm.Type,
) llvm.Value {
	return b.CreateGEP(
		b.CreateBitCast(
			v,
			llvm.PointerType(
				llvm.PointerType(
					llvm.FunctionType(t, []llvm.Type{types.NewEnvironment(0).LLVMPointerType()}, false),
					0,
				),
				0,
			),
			"",
		),
		[]llvm.Value{llvm.ConstIntFromString(llvm.Int32Type(), "-1", 10)},
		"",
	)
}
