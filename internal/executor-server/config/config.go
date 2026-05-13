package config

import (
	"fmt"
	"os"
	"path/filepath"
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

func hasWorkerClass(dir string) bool {
	workerClass := filepath.Join(dir, "Worker.class")

	info, err := os.Stat(workerClass)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func findWorkerFromRoot(root string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(rootAbs, "internal", "executor", "worker"),
		rootAbs,
	}

	for _, candidate := range candidates {
		if hasWorkerClass(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("Worker.class não encontrado a partir de %s", rootAbs)
}

func findWorkerClasspath() (string, error) {
	if classpath := os.Getenv("EXECUTOR_JVM_CLASSPATH"); classpath != "" {
		classpathAbs, err := filepath.Abs(classpath)
		if err != nil {
			return "", err
		}

		if hasWorkerClass(classpathAbs) {
			return classpathAbs, nil
		}
	}

	if root := os.Getenv("EXECUTOR_ROOT_DIR"); root != "" {
		workerDir, err := findWorkerFromRoot(root)
		if err == nil {
			return workerDir, nil
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		workerDir := filepath.Join(dir, "internal", "executor", "worker")

		if hasWorkerClass(workerDir) {
			return workerDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return "", fmt.Errorf("Worker.class não encontrado a partir do diretório atual")
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

	classpath, err := findWorkerClasspath()
	if err != nil {
		panic("não foi possível localizar o classpath do Worker JVM: " + err.Error())
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
