package trader

import "nofx/logger"

func (pt *PredictionTrader) logInfof(format string, args ...interface{}) {
	logger.Infof("[prediction:%s] "+format, append([]interface{}{pt.name}, args...)...)
}

func (pt *PredictionTrader) logWarnf(format string, args ...interface{}) {
	logger.Warnf("[prediction:%s] "+format, append([]interface{}{pt.name}, args...)...)
}
