package workerPool

// Запрос на обработку воркер пулом
type Request struct {
	// функция, которую выполнит воркер
	F func()
	// тип запрос
	RType
}

type RType byte

const (
	// бизнес-задача
	RTypeBusiness = iota
	// задача на завершение работы пула, тк закончились задачи
	RTypeTasksEnded
)
