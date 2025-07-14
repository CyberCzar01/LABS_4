package evaluator

// описывает удалённые операции с роботом
type RobotClient interface {
	Step(dx, dy, dz int) (bool, error)
	Look(dx, dy, dz int) (int, error)
	Pos() (int, int, int, error)
	Break() error
}

var client RobotClient

// задаёт клиента для операций с роботом
func InitRobot(c RobotClient) {
	client = c
}
