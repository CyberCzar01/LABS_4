package evaluator

import (
	"drone-maze/ast"
	"drone-maze/maze"
	"drone-maze/object"
	"fmt"
	"strings"
)

var (
	NULL         = &object.Null{}
	SHINRI       = &object.Boolean{Value: true}
	USO          = &object.Boolean{Value: false}
	BREAK        = &object.BreakValue{}
	CONTINUE     = &object.ContinueValue{}
	mazeInstance *maze.Maze
)

func InitState(m *maze.Maze) {
	mazeInstance = m
}

func GetMaze() *maze.Maze {
	return mazeInstance
}

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env)
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.HexLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)
	case *ast.BlockStatement:
		return evalBlockStatement(node, env)
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}
	case *ast.VariableDeclaration:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		declared := strings.ToUpper(node.Type)
		if declared != string(val.Type()) {
			return newError("type mismatch: cannot assign %s to %s", val.Type(), declared)
		}
		env.Set(node.Name.Value, val)
		return nil
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Env: env, Body: body}
	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args)
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)
	case *ast.SetExpression:
		return evalSetExpression(node, env)
	case *ast.LoopExpression:
		return evalLoopExpression(node, env)
	case *ast.BreakStatement:
		return BREAK
	case *ast.ContinueStatement:
		return CONTINUE
	case *ast.CellLiteral:
		return evalCellLiteral(node, env)
	case *ast.FieldAccessExpression:
		obj := Eval(node.Object, env)
		if isError(obj) {
			return obj
		}
		return evalFieldAccessExpression(obj, node.FieldName)
	case *ast.RuikeiExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		if node.Right == nil {
			return &object.String{Value: string(left.Type())}
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		if left.Type() == right.Type() {
			return SHINRI
		}
		return USO
	case *ast.JigenExpression:
		return evalJigenExpression(node, env)
	case *ast.RobotOperation:
		return evalRobotOperation(node, env)
	case *ast.FunctionDeclaration:
		fn := &object.Function{Parameters: node.Parameters, Body: node.Body, Env: env}
		env.Set(node.Name.Value, fn)
		return nil
	case *ast.AssignmentExpression:
		return evalAssignmentExpression(node, env)
	case *ast.ArrayDimensionsLiteral:
		return evalArrayDimensionsLiteral(node, env)
	case *ast.MultiIndexExpression:
		return evalMultiIndexExpression(node, env)
	case *ast.SequenceExpression:
		return evalSequenceExpression(node, env)
	}

	return nil
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch r := result.(type) {
		case *object.ReturnValue:
			return r.Value
		case *object.Error:
			return r
		case *object.BreakValue:
			result = NULL
			continue
		}
	}

	return result
}

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object
	for _, statement := range block.Statements {
		result = Eval(statement, env)
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ || rt == object.BREAK_OBJ || rt == object.CONTINUE_OBJ {
				return result
			}
		}
	}
	return result
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return SHINRI
	}
	return USO
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "~":
		return evalNotOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalNotOperatorExpression(right object.Object) object.Object {
	switch right {
	case SHINRI:
		return USO
	case USO:
		return SHINRI
	case NULL:
		return SHINRI
	default:
		if right.Type() == object.INTEGER_OBJ {
			if right.(*object.Integer).Value != 0 {
				return USO // a truthy value becomes false
			}
			return SHINRI
		}
		return newError("unknown operator: ~%s", right.Type())
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}
	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.INTEGER_OBJ:
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return evalBooleanInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value
	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		if rightVal == 0 {
			return newError("division by zero")
		}
		return &object.Integer{Value: leftVal / rightVal}
	case "%":
		if rightVal == 0 {
			return newError("division by zero")
		}
		return &object.Integer{Value: leftVal % rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalBooleanInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Boolean).Value
	rightVal := right.(*object.Boolean).Value
	switch operator {
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "^":
		return nativeBoolToBooleanObject(leftVal && rightVal)
	case "v":
		return nativeBoolToBooleanObject(leftVal || rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}
	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func evalLoopExpression(le *ast.LoopExpression, env *object.Environment) object.Object {
	startVal := Eval(le.Start, env)
	if isError(startVal) {
		return startVal
	}
	startInt, ok := startVal.(*object.Integer)
	if !ok {
		return newError("loop start must be an integer, got %s", startVal.Type())
	}

	endVal := Eval(le.End, env)
	if isError(endVal) {
		return endVal
	}
	endInt, ok := endVal.(*object.Integer)
	if !ok {
		return newError("loop end must be an integer, got %s", endVal.Type())
	}

	oldVal, hadOld := env.Get(le.Variable.Value)
	var result object.Object

	for i := startInt.Value; i < endInt.Value; i++ {
		env.Set(le.Variable.Value, &object.Integer{Value: i})
		result = Eval(le.Body, env)

		if result != nil {
			switch result.Type() {
			case object.BREAK_OBJ:
				result = NULL
				goto loopEnd
			case object.CONTINUE_OBJ:
				continue
			case object.RETURN_VALUE_OBJ, object.ERROR_OBJ:
				goto loopEnd
			}
		}
	}
loopEnd:

	if hadOld {
		env.Set(le.Variable.Value, oldVal)
	} else {
		env.Delete(le.Variable.Value)
	}

	return result
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case SHINRI:
		return true
	case USO:
		return false
	}
	if i, ok := obj.(*object.Integer); ok {
		return i.Value != 0
	}
	return true
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	return newError("identifier not found: " + node.Value)
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object
	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	function, ok := fn.(*object.Function)
	if !ok {
		return newError("not a function: %s", fn.Type())
	}

	if len(function.Parameters) != len(args) {
		return newError("wrong number of arguments: want=%d, got=%d",
			len(function.Parameters), len(args))
	}

	extendedEnv := extendFunctionEnv(function, args)
	evaluated := Eval(function.Body, extendedEnv)
	return unwrapReturnValue(evaluated)
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)
	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}
	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		arr := left.(*object.Array)
		if len(arr.Dimensions) > 1 {
			return newError("incorrect number of indexes")
		}
		return evalArrayIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arr := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arr.Elements))

	if idx < 0 || idx >= max {
		return newError("index out of bounds: %d", idx)
	}
	return arr.Elements[idx]
}

func evalCellLiteral(node *ast.CellLiteral, env *object.Environment) object.Object {
	x := Eval(node.X, env)
	if isError(x) {
		return x
	}
	y := Eval(node.Y, env)
	if isError(y) {
		return y
	}
	z := Eval(node.Z, env)
	if isError(z) {
		return z
	}
	isObstacle := Eval(node.IsObstacle, env)
	if isError(isObstacle) {
		return isObstacle
	}
	return &object.Cell{X: x, Y: y, Z: z, IsObstacle: isObstacle}
}

func evalFieldAccessExpression(obj object.Object, fieldName *ast.Identifier) object.Object {
	cell, ok := obj.(*object.Cell)
	if !ok {
		return newError("field access requires a rippotai, got %s", obj.Type())
	}

	switch fieldName.Value {
	case "x":
		return cell.X
	case "y":
		return cell.Y
	case "z":
		return cell.Z
	case "is_obstacle":
		if b, ok := cell.IsObstacle.(*object.Boolean); ok {
			return b
		}
		return newError("is_obstacle field is not a boolean, got %T", cell.IsObstacle)
	case "is_exit":
		if cell.IsExit == nil {
			return USO
		}
		if b, ok := cell.IsExit.(*object.Boolean); ok {
			return b
		}
		return newError("is_exit field is not a boolean, got %T", cell.IsExit)
	default:
		return newError("rippotai has no field named %s", fieldName.Value)
	}
}

func evalRobotOperation(op *ast.RobotOperation, env *object.Environment) object.Object {
	// Если настроен внешний клиент – используем его
	if client != nil {
		switch op.Type {
		case "^_^":
			client.Step(0, 1, 0)
			return NULL
		case "v_v":
			client.Step(0, -1, 0)
			return NULL
		case "<_<":
			client.Step(-1, 0, 0)
			return NULL
		case ">_>":
			client.Step(1, 0, 0)
			return NULL
		case "o_o":
			client.Step(0, 0, 1)
			return NULL
		case "~_~":
			client.Step(0, 0, -1)
			return NULL
		case "*_*":
			x, y, z, _ := client.Pos()
			return &object.Cell{X: &object.Integer{Value: int64(x)}, Y: &object.Integer{Value: int64(y)}, Z: &object.Integer{Value: int64(z)}, IsObstacle: USO}
		case "^_0":
			d, _ := client.Look(0, 1, 0)
			return &object.Integer{Value: int64(d)}
		case "v_0":
			d, _ := client.Look(0, -1, 0)
			return &object.Integer{Value: int64(d)}
		case "<_0":
			d, _ := client.Look(-1, 0, 0)
			return &object.Integer{Value: int64(d)}
		case ">_0":
			d, _ := client.Look(1, 0, 0)
			return &object.Integer{Value: int64(d)}
		case "o_0":
			d, _ := client.Look(0, 0, 1)
			return &object.Integer{Value: int64(d)}
		case "~_0":
			d, _ := client.Look(0, 0, -1)
			return &object.Integer{Value: int64(d)}
		case ">_<":
			client.Break()
			return BREAK
		default:
			return newError("unknown robot operation: %s", op.Type)
		}
	}

	// Для тестов
	if mazeInstance == nil {
		return newError("no maze loaded, robot operations are disabled")
	}

	robot := mazeInstance.GetRobot()
	if robot.IsBroken {
		return newError("robot is broken")
	}

	switch op.Type {
	case "^_^":
		mazeInstance.Move(0, 1, 0)
		return NULL
	case "v_v":
		mazeInstance.Move(0, -1, 0)
		return NULL
	case "<_<":
		mazeInstance.Move(-1, 0, 0)
		return NULL
	case ">_>":
		mazeInstance.Move(1, 0, 0)
		return NULL
	case "o_o":
		mazeInstance.Move(0, 0, 1)
		return NULL
	case "~_~":
		mazeInstance.Move(0, 0, -1)
		return NULL
	case "*_*":
		mazeInstance.Draw()
		isExit := nativeBoolToBooleanObject(mazeInstance.IsExit(robot.Position.X, robot.Position.Y, robot.Position.Z))
		return &object.Cell{
			X:          &object.Integer{Value: int64(robot.Position.X)},
			Y:          &object.Integer{Value: int64(robot.Position.Y)},
			Z:          &object.Integer{Value: int64(robot.Position.Z)},
			IsObstacle: USO,
			IsExit:     isExit,
		}
	case "^_0":
		return &object.Integer{Value: int64(mazeInstance.Measure(0, 1, 0))}
	case "v_0":
		return &object.Integer{Value: int64(mazeInstance.Measure(0, -1, 0))}
	case "<_0":
		return &object.Integer{Value: int64(mazeInstance.Measure(-1, 0, 0))}
	case ">_0":
		return &object.Integer{Value: int64(mazeInstance.Measure(1, 0, 0))}
	case "o_0":
		return &object.Integer{Value: int64(mazeInstance.Measure(0, 0, 1))}
	case "~_0":
		return &object.Integer{Value: int64(mazeInstance.Measure(0, 0, -1))}
	case ">_<":
		return BREAK
	default:
		return newError("unknown robot operation: %s", op.Type)
	}
}

func evalSetExpression(node *ast.SetExpression, env *object.Environment) object.Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}

	switch target := node.Target.(type) {
	case *ast.IndexExpression:
		left := Eval(target.Left, env)
		if isError(left) {
			return left
		}

		index := Eval(target.Index, env)
		if isError(index) {
			return index
		}

		arr, ok := left.(*object.Array)
		if !ok {
			return newError("index operator cannot be applied to type %s", left.Type())
		}

		idx, ok := index.(*object.Integer)
		if !ok {
			return newError("index must be an INTEGER, got %s", index.Type())
		}

		if idx.Value < 0 || idx.Value >= int64(len(arr.Elements)) {
			return newError("index out of bounds: %d", idx.Value)
		}

		arr.Elements[idx.Value] = val
		return val
	case *ast.MultiIndexExpression:
		left := Eval(target.Left, env)
		if isError(left) {
			return left
		}
		arr, ok := left.(*object.Array)
		if !ok {
			return newError("index operator cannot be applied to type %s", left.Type())
		}
		idxVals := []int64{}
		for _, idxExpr := range target.Indexes {
			idxObj := Eval(idxExpr, env)
			if isError(idxObj) {
				return idxObj
			}
			intIdx, ok := idxObj.(*object.Integer)
			if !ok {
				return newError("index must be INTEGER, got %s", idxObj.Type())
			}
			idxVals = append(idxVals, intIdx.Value)
		}
		if len(idxVals) != len(arr.Dimensions) {
			return newError("incorrect number of indexes")
		}
		off, err := calcOffset(arr.Dimensions, idxVals)
		if err != nil {
			return newError(err.Error())
		}
		arr.Elements[off] = val
		return val
	default:
		return newError("invalid target for SET expression: %T", target)
	}
}

func evalJigenExpression(node *ast.JigenExpression, env *object.Environment) object.Object {
	arrObj := Eval(node.Array, env)
	if isError(arrObj) {
		return arrObj
	}
	array, ok := arrObj.(*object.Array)
	if !ok {
		return newError("jigen expects an array, got %s", arrObj.Type())
	}
	dims := int64(1)
	if len(array.Dimensions) > 0 {
		dims = int64(len(array.Dimensions))
	}
	return &object.Integer{Value: dims}
}

func evalAssignmentExpression(node *ast.AssignmentExpression, env *object.Environment) object.Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}

	existing, ok := env.Get(node.Name.Value)
	if !ok {
		return newError("identifier not found: %s", node.Name.Value)
	}

	if existing.Type() != val.Type() {
		return newError("type mismatch: cannot assign %s to %s", val.Type(), existing.Type())
	}

	env.Update(node.Name.Value, val)
	return val
}

func evalArrayDimensionsLiteral(adl *ast.ArrayDimensionsLiteral, env *object.Environment) object.Object {
	dims := []int64{}
	total := int64(1)
	for _, expr := range adl.Dims {
		valObj := Eval(expr, env)
		if isError(valObj) {
			return valObj
		}
		intObj, ok := valObj.(*object.Integer)
		if !ok {
			return newError("array dimension must be integer, got %s", valObj.Type())
		}
		if intObj.Value <= 0 {
			return newError("array dimension must be positive, got %d", intObj.Value)
		}
		dims = append(dims, intObj.Value)
		total *= intObj.Value
	}
	elements := make([]object.Object, total)
	for i := range elements {
		elements[i] = NULL
	}
	return &object.Array{Elements: elements, Dimensions: dims}
}

func evalMultiIndexExpression(node *ast.MultiIndexExpression, env *object.Environment) object.Object {
	leftObj := Eval(node.Left, env)
	if isError(leftObj) {
		return leftObj
	}
	arr, ok := leftObj.(*object.Array)
	if !ok {
		return newError("index operator not supported: %s", leftObj.Type())
	}
	if len(node.Indexes) != len(arr.Dimensions) {
		return newError("incorrect number of indexes: expected %d, got %d", len(arr.Dimensions), len(node.Indexes))
	}
	idxVals := []int64{}
	for _, idxExpr := range node.Indexes {
		idxObj := Eval(idxExpr, env)
		if isError(idxObj) {
			return idxObj
		}
		intIdx, ok := idxObj.(*object.Integer)
		if !ok {
			return newError("index must be integer, got %s", idxObj.Type())
		}
		idxVals = append(idxVals, intIdx.Value)
	}
	offset, err := calcOffset(arr.Dimensions, idxVals)
	if err != nil {
		return newError(err.Error())
	}
	return arr.Elements[offset]
}

func calcOffset(dims []int64, idx []int64) (int64, error) {
	offset := int64(0)
	stride := int64(1)
	for i := len(dims) - 1; i >= 0; i-- {
		if idx[i] < 0 || idx[i] >= dims[i] {
			return 0, fmt.Errorf("index out of bounds: %d (dim %d)", idx[i], dims[i])
		}
		offset += idx[i] * stride
		stride *= dims[i]
	}
	return offset, nil
}

func evalSequenceExpression(se *ast.SequenceExpression, env *object.Environment) object.Object {
	var last object.Object = NULL
	for _, op := range se.Operations {
		if ro, ok := op.(*ast.RobotOperation); ok && ro.Type == ">_<" {
			if intObj, ok := last.(*object.Integer); ok && intObj.Value == 1 {
				// условие выполнено – прерываем текущую последовательность
				return BREAK
			}
			continue
		}

		res := Eval(op, env)
		if isError(res) {
			return res
		}
		if res != nil && res.Type() == object.BREAK_OBJ {
			return NULL
		}
		if res != nil {
			last = res
		}
	}
	return last
}
