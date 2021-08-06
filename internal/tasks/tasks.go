package tasks

import "github.com/sirupsen/logrus"

// Pipeline is an implementation of a pipeline
type Pipeline interface {
	StartListener(listenerError chan error)
	Execute(string) []error
	Close() error
	GetCorrelationId() string
	GetLogger() *logrus.Logger
}

type PfPipeline struct {
}
