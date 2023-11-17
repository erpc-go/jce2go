package log

func Debug(f string, p ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(f, p...)
	}
}

func Info(f string, p ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Info(f, p...)
	}
}

func Warn(f string, p ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(f, p...)
	}
}

func Error(f string, p ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Error(f, p...)
	}
}

func Panic(f string, p ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Panic(f, p...)
	}
}

func Fatal(f string, p ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Fatal(f, p...)
	}
}

func Raw(f string, p ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Raw(f, p...)
	}
}

func SetDefaultLogger(logger *Logger) {
	DefaultLogger = logger
	DefaultLogger.depth = 3
}
