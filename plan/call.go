package plan

import (
	"fmt"
	"github.com/viant/igo/exec"
	"github.com/viant/igo/exec/est"
	"github.com/viant/xunsafe"
	"go/ast"
	"reflect"
	"strings"
)

func (s *Scope) compileCallExpr(callExpr *ast.CallExpr) (est.New, []reflect.Type, error) {
	id := stringifyExpr(callExpr.Fun, 0)
	fnNew, fType, ok, err := s.compileBuildIn(id, callExpr)
	if ok {

		return fnNew, []reflect.Type{fType}, err
	}
	aCaller, args, retTypes, err := s.compileCallExprSignature(callExpr)
	if err != nil {
		return nil, nil, err
	}
	newFn, err := est.NewCaller(aCaller, args)
	if err != nil {
		return nil, nil, err
	}
	return newFn, retTypes, nil
}

func (s *Scope) compileCallExprAssign(callExpr *ast.CallExpr, dest []*est.Operand) (est.New, error) {
	id := stringifyExpr(callExpr.Fun, 0)
	fnNew, fType, ok, err := s.compileBuildIn(id, callExpr)
	if ok {
		_ = s.adjust(dest[0].Selector, fType)
		return est.NewAssignExpr(dest[0], est.NewOperand(nil, fType, fnNew, nil)), nil
	}
	caller, args, ret, err := s.compileCallExprSignature(callExpr)
	if err != nil {
		return nil, err
	}

	for i := range dest {
		if ret[i].Kind() == reflect.Interface {
			if dest[i].Type != nil {
				continue
			}
		}
		if err = s.adjust(dest[i].Selector, ret[i]); err != nil {
			return nil, err
		}
		dest[i].Type = xunsafe.NewType(ret[i])
	}
	return est.NewCallExprAssign(caller, args, dest)
}

//func (s *Scope) compileSingleReturnTypeCallExpr(callExpr *ast.CallExpr) (est.New, reflect.Type, error) {
//	id := stringifyExpr(callExpr.Fun, 0)
//	fnNew, fType, ok, err := s.compileBuildIn(id, callExpr)
//	if ok {
//		return fnNew, fType, err
//	}
//	newFn, retTypes, err := s.compileCallExpr(callExpr)
//	if err != nil {
//		return nil, nil, err
//	}
//	if len(retTypes) == 0 {
//		return newFn, nil, nil
//	}
//	return newFn, retTypes[0], nil
//}

func (s *Scope) compileCallExprSignature(callExpr *ast.CallExpr) (exec.Caller, []*est.Operand, []reflect.Type, error) {
	id := stringifyExpr(callExpr.Fun, 0)
	var ok bool
	holder, fn, err := s.discoverMethod(id)
	if err != nil {
		return nil, nil, nil, err
	}
	if fn == nil {
		fn, ok = s.lookupFunction(id)
		if !ok {
			return nil, nil, nil, fmt.Errorf("failed to lookup function: %v", id)
		}
	}

	args, err := s.compileCallArgs(callExpr, holder)
	if err != nil {
		return nil, nil, nil, err
	}
	funType := reflect.TypeOf(fn)
	var retTypes = make([]reflect.Type, funType.NumOut())
	for i := range retTypes {
		retTypes[i] = funType.Out(i)
	}

	aCaller := asCaller(fn)
	if aCaller == nil {
		aCaller = newCaller(fn)
	}

	return aCaller, args, retTypes, nil
}

func (s *Scope) discoverMethod(id string) (*exec.Selector, interface{}, error) {
	holder, method, err := s.selectorFun(id)
	if err != nil || holder == nil {
		return nil, nil, err
	}
	fn, err := s.lookupMethod(holder, method)
	return holder, fn, err
}

func (s *Scope) selectorFun(id string) (*exec.Selector, string, error) {
	pos := strings.LastIndex(id, ".")
	if pos == -1 {
		return nil, id, nil
	}
	holderName := id[:pos]
	fName := id[pos+1:]
	sel, err := s.Selector(holderName)
	if err != nil {
		return nil, "", err
	}
	return sel, fName, nil
}

func (s *Scope) compileCallArgs(callExpr *ast.CallExpr, holder *exec.Selector) ([]*est.Operand, error) {
	argLength := len(callExpr.Args)
	if holder != nil {
		argLength++
	}
	var args = make([]*est.Operand, argLength)
	if len(args) == 0 {
		return args, nil
	}
	var err error
	i := 0
	if holder != nil {
		args[i] = est.NewOperand(holder, nil, nil, nil)
		i++
	}
	for _, expr := range callExpr.Args {
		if args[i], err = s.assembleOperand(expr, false); err != nil {
			return nil, err
		}
		i++
	}
	return args, nil
}

func (s *Scope) lookupMethod(sel *exec.Selector, name string) (interface{}, error) {
	method, ok := sel.Type.MethodByName(name)
	if !ok {
		return nil, fmt.Errorf("failed to locate %v.%v", sel.Type.String(), name)
	}
	return method.Func.Interface(), nil
}
