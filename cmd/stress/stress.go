package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"Algorithms/internal/executor"
)

func geraEntrada(n int) (input string, expectedSum int) {
	input = fmt.Sprintf("%d\n", n)
	sum := 0
	for i := 0; i < n; i++ {
		num := rand.Intn(1000) + 1
		if i > 0 {
			input += " "
		}
		input += fmt.Sprintf("%d", num)
		sum += num
	}
	input += "\n"
	return input, sum
}

func main() {
	tamanho := 1000

	entrada, somaEsperada := geraEntrada(tamanho)
	expected := fmt.Sprintf("%d\n", somaEsperada)

	testCases := []executor.TestCase{
		{Input: entrada, Expected: expected},
	}

	code := `
n = int(input())
arr = list(map(int, input().split()))
# Bubble sort para aumentar a carga
for i in range(n):
    for j in range(n-1):
        if arr[j] > arr[j+1]:
            arr[j], arr[j+1] = arr[j+1], arr[j]
print(sum(arr))
`

	lang := "python"
	timeLimit := 10
	memLimit := 256

	const numSubmissions = 20

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numSubmissions; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			res, err := executor.Execute(ctx, code, lang, testCases, timeLimit, memLimit)
			if err != nil {
				fmt.Printf("[%02d] ERRO: %v\n", id, err)
				return
			}
			fmt.Printf("[%02d] Veredito: %s, tempo execução: %.2fs\n", id, res.Verdict, res.TotalTime)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("\nTempo total para %d submissões concorrentes: %.2fs\n", numSubmissions, elapsed)
}
