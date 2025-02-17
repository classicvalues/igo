package plan

import (
	"fmt"
	"github.com/viant/igo/exec"
	"github.com/viant/igo/exec/est"
	"go/ast"
	"reflect"
)

func (s *Scope) compileReducer(holder *exec.Selector, expr *ast.CallExpr) (est.New, reflect.Type, error) {
	args := expr.Args
	if len(args) == 0 {
		return nil, nil, fmt.Errorf("%v insuficient arguments", stringifyExpr(expr.Fun, 0))
	}
	funcLit, ok := args[0].(*ast.FuncLit)
	if !ok {
		return nil, nil, fmt.Errorf("%v invalid args", stringifyExpr(expr.Fun, 0))
	}
	var init *est.Operand
	var err error
	if len(args) > 1 {
		init, err = s.assembleOperand(args[1], false)
	}
	scope := s.subScope()
	parameters, err := scope.paramSelectors(funcLit.Type.Params.List)
	if err != nil {
		return nil, nil, err
	}
	results, err := scope.paramSelectors(funcLit.Type.Results.List)
	if err != nil {
		return nil, nil, err
	}
	scope.out = &results
	body, err := scope.compileBlockStmt(funcLit.Body, false)
	return est.NewReducer(holder, parameters, results, init, body)
}
