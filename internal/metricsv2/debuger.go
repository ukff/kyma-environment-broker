package metricsv2

import (
	`fmt`
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	m sync.Mutex
)

func Debug(logger logrus.FieldLogger, source, message string) {
	m.Lock()
	defer m.Unlock()
	logger.Infof("#Debug: from %s -> %s", source, message)
	fmt.Println(fmt.Sprintf("#Debug: from %s -> %s", source, message))
}
