package common_interfaces

import (
	"github.com/sirupsen/logrus"
)

type PipelineInterface interface {
	GetCorrelationID() string
	SetCorrelationID(string)
	Execute(string) []error
	StartListener(chan error)
	GetLogger() *logrus.Entry
	SetLogger(*logrus.Entry)
	Close() error
}
