package utils

type IServer interface {
	Start() error
	Stop()
}
