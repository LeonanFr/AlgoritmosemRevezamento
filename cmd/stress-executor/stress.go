package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"
)

type ExecuteRequest struct {
	Code      string   `json:"code"`
	Language  string   `json:"language"`
	Inputs    []string `json:"inputs"`
	Expected  []string `json:"expected"`
	TimeLimit int      `json:"timeLimit,omitempty"`
	MemLimit  int      `json:"memLimit,omitempty"`
	Mode      string   `json:"mode,omitempty"`
}

type ExecuteResponse struct {
	Verdict   string  `json:"verdict"`
	Message   string  `json:"message,omitempty"`
	TotalTime float64 `json:"totalTime,omitempty"`
}

const kotlinCode = `fun bubbleSort(arr: IntArray) {
    val n = arr.size
    for (i in 0 until n) {
        for (j in 0 until n-1) {
            if (arr[j] > arr[j+1]) {
                val temp = arr[j]
                arr[j] = arr[j+1]
                arr[j+1] = temp
            }
        }
    }
}
fun main() {
    val n = readLine()!!.toInt()
    val arr = readLine()!!.split(" ").map { it.toInt() }.toIntArray()
    bubbleSort(arr)
    println(arr.joinToString(" "))
}`

func generateTestCases(numCases int, arraySize int) (inputs []string, expected []string) {
	inputs = make([]string, numCases)
	expected = make([]string, numCases)

	for i := 0; i < numCases; i++ {
		nums := make([]int, arraySize)
		for j := 0; j < arraySize; j++ {
			nums[j] = rand.Intn(1000) + 1
		}

		sorted := make([]int, arraySize)
		copy(sorted, nums)
		sort.Ints(sorted)

		input := fmt.Sprintf("%d\n", arraySize)
		for j, num := range nums {
			if j > 0 {
				input += " "
			}
			input += fmt.Sprintf("%d", num)
		}
		input += "\n"

		exp := ""
		for j, num := range sorted {
			if j > 0 {
				exp += " "
			}
			exp += fmt.Sprintf("%d", num)
		}
		exp += "\n"

		inputs[i] = input
		expected[i] = exp
	}
	return
}

func main() {
	url := "http://localhost:8081/execute"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJleGVjdXRvciIsInJvbGUiOiJzeXN0ZW0ifQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	const numRequests = 5
	var wg sync.WaitGroup
	startGlobal := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			numCases := 15
			arraySize := 300
			var inputs, expected []string

			inputs, expected = generateTestCases(numCases, arraySize)

			reqBody := ExecuteRequest{
				Code:      kotlinCode,
				Language:  "kotlin",
				Inputs:    inputs,
				Expected:  expected,
				Mode:      "submit",
				TimeLimit: 1,
				MemLimit:  1024,
			}
			jsonData, _ := json.Marshal(reqBody)

			client := http.Client{Timeout: 5 * time.Minute}
			req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonData))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			startReq := time.Now()
			resp, err := client.Do(req)
			elapsedReq := time.Since(startReq).Seconds()

			if err != nil {
				fmt.Printf("[%02d] Erro após %.2fs: %v\n", id, elapsedReq, err)
				return
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			var result ExecuteResponse
			_ = json.Unmarshal(body, &result)

			fmt.Printf("[%02d] Status: %s, veredito: %s, tempo: %.2fs, casos=%d, tamanho=%d\n",
				id, resp.Status, result.Verdict, elapsedReq, numCases, arraySize)
		}(i)
	}

	wg.Wait()
	elapsedGlobal := time.Since(startGlobal).Seconds()
	fmt.Printf("Tempo total para %d submissões Kotlin: %.2fs\n", numRequests, elapsedGlobal)
}
