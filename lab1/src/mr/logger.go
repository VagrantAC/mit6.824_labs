package mr

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func init() {
	// 生产环境使用 JSON 格式，开发环境使用文本格式
	if os.Getenv("ENV") == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
			DisableHTMLEscape: true,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat:  "2006-01-02 15:04:05",
			FullTimestamp:    true,
			ForceColors:      true,
			DisableColors:    false,
			QuoteEmptyFields: true,
			SortingFunc: func(fields []string) {
				// 将关键字段放在前面
				priority := map[string]int{
					"service":   0,
					"component": 1,
					"worker_id": 2,
					"task_id":   3,
					"func":      4,
				}

				// 自定义排序
				for i := 0; i < len(fields); i++ {
					for j := i + 1; j < len(fields); j++ {
						p1, ok1 := priority[fields[i]]
						p2, ok2 := priority[fields[j]]
						if ok1 && !ok2 {
							continue
						}
						if !ok1 && ok2 {
							fields[i], fields[j] = fields[j], fields[i]
						}
						if ok1 && ok2 && p1 > p2 {
							fields[i], fields[j] = fields[j], fields[i]
						}
					}
				}
			},
		})
	}

	// 从环境变量读取日志级别
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	if logLevel, err := logrus.ParseLevel(level); err == nil {
		logger.SetLevel(logLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.SetOutput(os.Stdout)
}

func GetLogger() *logrus.Logger {
	return logger
}

func WithField(key string, value interface{}) *logrus.Entry {
	return logger.WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return logger.WithFields(fields)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

func Warn(args ...interface{}) {
	logger.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}
