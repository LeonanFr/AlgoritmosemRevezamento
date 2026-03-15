package executor

import (
	"runtime"
)

type Language struct {
	Name        string
	Extension   string
	CompileCmd  []string
	RunCmd      []string
	NeedCompile bool
}

var languages map[string]Language

func init() {
	base := map[string]Language{
		"c": {
			Name:        "C",
			Extension:   ".c",
			CompileCmd:  []string{"gcc", "code.c", "-o", "code"},
			RunCmd:      []string{"./code"},
			NeedCompile: true,
		},
		"cpp": {
			Name:        "C++",
			Extension:   ".cpp",
			CompileCmd:  []string{"g++", "code.cpp", "-o", "code"},
			RunCmd:      []string{"./code"},
			NeedCompile: true,
		},
		"python": {
			Name:        "Python",
			Extension:   ".py",
			CompileCmd:  nil,
			RunCmd:      []string{"python3", "code.py"},
			NeedCompile: false,
		},
		"java": {
			Name:        "Java",
			Extension:   ".java",
			CompileCmd:  []string{"javac", "Main.java"},
			RunCmd:      []string{"java", "-cp", ".", "Main"},
			NeedCompile: true,
		},
		"kotlin": {
			Name:        "Kotlin",
			Extension:   ".kt",
			CompileCmd:  []string{"kotlinc", "code.kt", "-include-runtime", "-d", "code.jar"},
			RunCmd:      []string{"java", "-jar", "code.jar"},
			NeedCompile: true,
		},
		"javascript": {
			Name:        "JavaScript",
			Extension:   ".js",
			CompileCmd:  nil,
			RunCmd:      []string{"node", "code.js"},
			NeedCompile: false,
		},
	}

	languages = make(map[string]Language)
	for k, v := range base {
		languages[k] = v
	}

	if runtime.GOOS == "windows" {
		if py, ok := languages["python"]; ok {
			py.RunCmd = []string{"python", "code.py"}
			languages["python"] = py
		}
		if c, ok := languages["c"]; ok {
			c.CompileCmd = []string{"gcc", "code.c", "-o", "code.exe"}
			c.RunCmd = []string{"code.exe"}
			languages["c"] = c
		}
		if cpp, ok := languages["cpp"]; ok {
			cpp.CompileCmd = []string{"g++", "code.cpp", "-o", "code.exe"}
			cpp.RunCmd = []string{"code.exe"}
			languages["cpp"] = cpp
		}
	}
}
