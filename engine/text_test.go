package engine

import (
	"context"
	"embed"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed testdata
var testdata embed.FS

func TestVM_Compile(t *testing.T) {
	tests := []struct {
		title  string
		text   string
		args   []interface{}
		err    error
		result map[procedureIndicator]procedure
	}{
		{title: "shebang", text: `#!/foo/bar
foo(a).
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}}, xrTable: []Term{NewAtom("a")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "shebang: no following lines", text: `#!/foo/bar`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				multifile: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}}, xrTable: []Term{NewAtom("c")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "facts", text: `
foo(a).
foo(b).
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}}, xrTable: []Term{NewAtom("a")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("b")}}, xrTable: []Term{NewAtom("b")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "rules", text: `
bar(X) :- foo(X).
baz(X) :- bar(X).
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				multifile: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}}, xrTable: []Term{NewAtom("c")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
			{name: NewAtom("bar"), arity: 1}: &userDefined{
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("bar"), arity: 1}, raw: atomIf.Apply(NewAtom("bar").Apply(NewNamedVariable("X")), NewAtom("foo").Apply(NewNamedVariable("X"))), xrTable: []Term{procedureIndicator{name: NewAtom("foo"), arity: 1}}, vars: []Variable{NewNamedVariable("X")}, bytecode: bytecode{
						{opcode: opVar, operand: 0},
						{opcode: opEnter},
						{opcode: opVar, operand: 0},
						{opcode: opCall, operand: 0},
						{opcode: opExit},
					}},
				},
			},
			{name: NewAtom("baz"), arity: 1}: &userDefined{
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("baz"), arity: 1}, raw: atomIf.Apply(NewAtom("baz").Apply(NewNamedVariable("X")), NewAtom("bar").Apply(NewNamedVariable("X"))), xrTable: []Term{procedureIndicator{name: NewAtom("bar"), arity: 1}}, vars: []Variable{NewNamedVariable("X")}, bytecode: bytecode{
						{opcode: opVar, operand: 0},
						{opcode: opEnter},
						{opcode: opVar, operand: 0},
						{opcode: opCall, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "dynamic", text: `
:- dynamic(foo/1).
foo(a).
foo(b).
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				public:  true,
				dynamic: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}}, xrTable: []Term{NewAtom("a")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("b")}}, xrTable: []Term{NewAtom("b")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "multifile", text: `
:- multifile(foo/1).
foo(a).
foo(b).
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				multifile: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}}, xrTable: []Term{NewAtom("c")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}}, xrTable: []Term{NewAtom("a")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("b")}}, xrTable: []Term{NewAtom("b")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "discontiguous", text: `
:- discontiguous(foo/1).
foo(a).
bar(a).
foo(b).
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				discontiguous: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}}, xrTable: []Term{NewAtom("a")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("b")}}, xrTable: []Term{NewAtom("b")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
			{name: NewAtom("bar"), arity: 1}: &userDefined{
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("bar"), arity: 1}, raw: &compound{functor: NewAtom("bar"), args: []Term{NewAtom("a")}}, xrTable: []Term{NewAtom("a")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "include", text: `
:- include('testdata/foo').
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 0}: &userDefined{
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 0}, raw: NewAtom("foo"), bytecode: bytecode{
						{opcode: opExit},
					}},
				},
			},
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				multifile: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}}, xrTable: []Term{NewAtom("c")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "ensure_loaded", text: `
:- ensure_loaded('testdata/foo').
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 0}: &userDefined{
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 0}, raw: NewAtom("foo"), bytecode: bytecode{
						{opcode: opExit},
					}},
				},
			},
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				multifile: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}}, xrTable: []Term{NewAtom("c")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "initialization", text: `
:- initialization(foo(c)).
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				multifile: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}}, xrTable: []Term{NewAtom("c")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},
		{title: "predicate-backed directive", text: `
:- foo(c).
`, result: map[procedureIndicator]procedure{
			{name: NewAtom("foo"), arity: 1}: &userDefined{
				multifile: true,
				clauses: clauses{
					{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}}, xrTable: []Term{NewAtom("c")}, bytecode: bytecode{
						{opcode: opConst, operand: 0},
						{opcode: opExit},
					}},
				},
			},
		}},

		{title: "error: invalid argument", text: `
foo(?).
`, args: []interface{}{nil}, err: errors.New("can't convert to term: <invalid reflect.Value>")},
		{title: "error: syntax error", text: `
foo().
`, err: unexpectedTokenError{actual: Token{kind: tokenClose, val: ")"}}},
		{title: "error: expansion error", text: `
:- ensure_loaded('testdata/break_term_expansion').
foo(a).
`, err: Exception{term: NewAtom("ball")}},
		{title: "error: variable fact", text: `
X.
`, err: InstantiationError(nil)},
		{title: "error: variable rule", text: `
X :- X.
`, err: InstantiationError(nil)},
		{title: "error: non-callable rule body", text: `
foo :- 1.
`, err: TypeError(ValidTypeCallable, Integer(1), nil)},
		{title: "error: non-PI argument", text: `
:- dynamic(foo).
`, err: TypeError(ValidTypePredicateIndicator, NewAtom("foo"), nil)},
		{title: "error: included variable", text: `
:- include(X).
`, err: InstantiationError(nil)},
		{title: "error: included file not found", text: `
:- include('testdata/not_found').
`, err: ExistenceError(ObjectTypeSourceSink, NewAtom("testdata/not_found"), nil)},
		{title: "error: included non-atom", text: `
:- include(1).
`, err: TypeError(ValidTypeAtom, Integer(1), nil)},
		{title: "error: initialization exception", text: `
:- initialization(bar).
`, err: ExistenceError(ObjectTypeProcedure, atomSlash.Apply(NewAtom("bar"), Integer(0)), nil)},
		{title: "error: initialization failure", text: `
:- initialization(foo(d)).
`, err: errors.New("failed initialization goal: foo(d)")},
		{title: "error: predicate-backed directive exception", text: `
:- bar.
`, err: ExistenceError(ObjectTypeProcedure, atomSlash.Apply(NewAtom("bar"), Integer(0)), nil)},
		{title: "error: predicate-backed directive failure", text: `
:- foo(d).
`, err: errors.New("failed directive: foo(d)")},
		{title: "error: discontiguous, end of text", text: `
foo(a).
bar(a).
foo(b).
`, err: &discontiguousError{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}}},
		{title: "error: discontiguous, before directive", text: `
foo(a).
bar(a).
foo(b).
:- foo(c).
`, err: &discontiguousError{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}}},
		{title: "error: discontiguous, before other facts", text: `
foo(a).
bar(a).
foo(b).
bar(b).
`, err: &discontiguousError{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			var vm VM
			vm.operators.define(1200, operatorSpecifierXFX, atomIf)
			vm.operators.define(1200, operatorSpecifierXFX, atomArrow)
			vm.operators.define(1200, operatorSpecifierFX, atomIf)
			vm.operators.define(400, operatorSpecifierYFX, atomSlash)
			vm.procedures = map[procedureIndicator]procedure{
				{name: NewAtom("foo"), arity: 1}: &userDefined{
					multifile: true,
					clauses: clauses{
						{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}, raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}}, xrTable: []Term{NewAtom("c")}, bytecode: bytecode{
							{opcode: opConst, operand: 0},
							{opcode: opExit},
						}},
					},
				},
			}
			vm.FS = testdata
			vm.Register1(NewAtom("throw"), Throw)
			assert.Equal(t, tt.err, vm.Compile(context.Background(), tt.text, tt.args...))
			if tt.err == nil {
				delete(vm.procedures, procedureIndicator{name: NewAtom("throw"), arity: 1})
				assert.Equal(t, tt.result, vm.procedures)
			}
		})
	}
}

func TestVM_Consult(t *testing.T) {
	tests := []struct {
		title string
		files Term
		ok    bool
		err   error
	}{
		{title: `:- consult('testdata/empty.txt').`, files: NewAtom("testdata/empty.txt"), ok: true},
		{title: `:- consult([]).`, files: List(), ok: true},
		{title: `:- consult(['testdata/empty.txt']).`, files: List(NewAtom("testdata/empty.txt")), ok: true},
		{title: `:- consult(['testdata/empty.txt', 'testdata/empty.txt']).`, files: List(NewAtom("testdata/empty.txt"), NewAtom("testdata/empty.txt")), ok: true},

		{title: `:- consult('testdata/abc.txt').`, files: NewAtom("testdata/abc.txt"), err: io.EOF},
		{title: `:- consult(['testdata/abc.txt']).`, files: List(NewAtom("testdata/abc.txt")), err: io.EOF},

		{title: `:- consult(X).`, files: NewNamedVariable("X"), err: InstantiationError(nil)},
		{title: `:- consult(foo(bar)).`, files: NewAtom("foo").Apply(NewAtom("bar")), err: TypeError(ValidTypeAtom, NewAtom("foo").Apply(NewAtom("bar")), nil)},
		{title: `:- consult(1).`, files: Integer(1), err: TypeError(ValidTypeAtom, Integer(1), nil)},
		{title: `:- consult(['testdata/empty.txt'|_]).`, files: PartialList(NewVariable(), NewAtom("testdata/empty.txt")), err: TypeError(ValidTypeAtom, PartialList(NewVariable(), NewAtom("testdata/empty.txt")), nil)},
		{title: `:- consult([X]).`, files: List(NewNamedVariable("X")), err: InstantiationError(nil)},
		{title: `:- consult([1]).`, files: List(Integer(1)), err: TypeError(ValidTypeAtom, Integer(1), nil)},

		{title: `:- consult('testdata/not_found.txt').`, files: NewAtom("testdata/not_found.txt"), err: ExistenceError(ObjectTypeSourceSink, NewAtom("testdata/not_found.txt"), nil)},
		{title: `:- consult(['testdata/not_found.txt']).`, files: List(NewAtom("testdata/not_found.txt")), err: ExistenceError(ObjectTypeSourceSink, NewAtom("testdata/not_found.txt"), nil)},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			vm := VM{
				FS: testdata,
			}
			ok, err := Consult(&vm, tt.files, Success, nil).Force(context.Background())
			assert.Equal(t, tt.ok, ok)
			if e, ok := tt.err.(Exception); ok {
				_, ok := NewEnv().Unify(e.Term(), err.(Exception).Term(), false)
				assert.True(t, ok)
			} else {
				assert.Equal(t, tt.err, err)
			}
		})
	}
}

func TestDiscontiguousError_Error(t *testing.T) {
	e := discontiguousError{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}}
	assert.Equal(t, "foo/1 is discontiguous", e.Error())
}
