package prewarm

import (
	"context"
	"log"
	"time"

	"Algorithms/internal/executor"
)

func LanguagesWarm() {
	languages := []string{"java", "kotlin"}
	for _, lang := range languages {
		go func(l string) {
			log.Printf("Pré-aquecendo %s...", l)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			var code string
			var testCase executor.TestCase
			switch l {
			case "java":
				code = `public class Main { public static void main(String[] args) { System.out.println(1); } }`
				testCase = executor.TestCase{Input: "", Expected: "1\n"}
			case "kotlin":
				code = `fun main() { println(1) }`
				testCase = executor.TestCase{Input: "", Expected: "1\n"}
			default:
				return
			}
			_, err := executor.Execute(ctx, code, l, []executor.TestCase{testCase}, 2, 256)
			if err != nil {
				log.Printf("Pré-aquecimento de %s falhou: %v", l, err)
			} else {
				log.Printf("Pré-aquecimento de %s concluído", l)
			}
		}(lang)
	}
}
