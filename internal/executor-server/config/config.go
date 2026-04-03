package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port             string
	AuthToken        string
	MaxConcurrent    int
	CasesParallel    int
	DefaultTimeLimit int
	DefaultMemLimit  int
	EnablePreWarm    bool
	JVMPoolSize      int
	JVMClasspath     string
	JVMXmx           string
}

func Load() *Config {
	port := os.Getenv("EXECUTOR_PORT")
	if port == "" {
		port = "8081"
	}
	token := os.Getenv("EXECUTOR_AUTH_TOKEN")
	if token == "" {
		token = "XYZ"
	}
	maxConc, _ := strconv.Atoi(os.Getenv("EXECUTOR_MAX_CONCURRENT"))
	if maxConc <= 0 {
		maxConc = 10
	}
	casesPar, _ := strconv.Atoi(os.Getenv("EXECUTOR_CASES_PARALLEL"))
	if casesPar <= 0 {
		casesPar = 4
	}
	defaultTime, _ := strconv.Atoi(os.Getenv("EXECUTOR_DEFAULT_TIME"))
	if defaultTime <= 0 {
		defaultTime = 5
	}
	defaultMem, _ := strconv.Atoi(os.Getenv("EXECUTOR_DEFAULT_MEM"))
	if defaultMem <= 0 {
		defaultMem = 256
	}
	preWarm, _ := strconv.ParseBool(os.Getenv("EXECUTOR_PREWARM"))
	poolSize, _ := strconv.Atoi(os.Getenv("EXECUTOR_JVM_POOL_SIZE"))
	if poolSize <= 0 {
		poolSize = 5
	}
	classpath := os.Getenv("EXECUTOR_JVM_CLASSPATH")
	if classpath == "" {
		classpath = "./internal/executor/worker"
	}
	jvmXmx := os.Getenv("EXECUTOR_JVM_XMX")
	if jvmXmx == "" {
		jvmXmx = "1g"
	}
	return &Config{
		Port:             port,
		AuthToken:        token,
		MaxConcurrent:    maxConc,
		CasesParallel:    casesPar,
		DefaultTimeLimit: defaultTime,
		DefaultMemLimit:  defaultMem,
		EnablePreWarm:    preWarm,
		JVMPoolSize:      poolSize,
		JVMClasspath:     classpath,
		JVMXmx:           jvmXmx,
	}
}
