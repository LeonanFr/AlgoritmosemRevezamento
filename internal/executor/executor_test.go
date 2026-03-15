package executor

import (
	"context"
	"testing"
)

func TestPythonAccepted(t *testing.T) {
	code := `print(sum(map(int, input().split())))`
	cases := []TestCase{
		{Input: "1 2\n", Expected: "3\n"},
		{Input: "10 20\n", Expected: "30\n"},
	}
	res, err := Execute(context.Background(), code, "python", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictAccepted {
		t.Errorf("esperado accepted, got %s", res.Verdict)
	}
	if len(res.TestCases) != 2 {
		t.Errorf("esperado 2 casos, got %d", len(res.TestCases))
	}
	for _, tc := range res.TestCases {
		if !tc.Passed {
			t.Errorf("caso %d falhou: output=%q esperado=%q", tc.Number, tc.Output, tc.Expected)
		}
	}
}

func TestPythonWrongAnswer(t *testing.T) {
	code := `print(42)`
	cases := []TestCase{
		{Input: "1 2\n", Expected: "3\n"},
	}
	res, err := Execute(context.Background(), code, "python", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictWrongAnswer {
		t.Errorf("esperado wrong_answer, got %s", res.Verdict)
	}
}

func TestPythonRuntimeError(t *testing.T) {
	code := `print(1/0)`
	cases := []TestCase{
		{Input: "", Expected: ""},
	}
	res, err := Execute(context.Background(), code, "python", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictRuntimeError {
		t.Errorf("esperado runtime_error, got %s", res.Verdict)
	}
}

func TestPythonTimeout(t *testing.T) {
	code := `while True: pass`
	cases := []TestCase{
		{Input: "", Expected: ""},
	}
	res, err := Execute(context.Background(), code, "python", cases, 1, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictTimeLimitExceeded {
		t.Errorf("esperado time_limit_exceeded, got %s", res.Verdict)
	}
}

func TestCAccepted(t *testing.T) {
	code := `#include <stdio.h>
int main() {
    int a, b;
    scanf("%d %d", &a, &b);
    printf("%d\n", a+b);
    return 0;
}`
	cases := []TestCase{
		{Input: "1 2\n", Expected: "3\n"},
		{Input: "10 20\n", Expected: "30\n"},
	}
	res, err := Execute(context.Background(), code, "c", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictAccepted {
		t.Errorf("esperado accepted, got %s", res.Verdict)
	}
}

func TestCCompilationError(t *testing.T) {
	code := `int main() { return 0; `
	cases := []TestCase{{Input: "", Expected: ""}}
	res, err := Execute(context.Background(), code, "c", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictCompilationError {
		t.Errorf("esperado compilation_error, got %s", res.Verdict)
	}
	if res.Message == "" {
		t.Error("mensagem de erro vazia")
	}
}

func TestCppAccepted(t *testing.T) {
	code := `#include <iostream>
using namespace std;
int main() {
    int a, b;
    cin >> a >> b;
    cout << a+b << endl;
    return 0;
}`
	cases := []TestCase{
		{Input: "1 2\n", Expected: "3\n"},
		{Input: "10 20\n", Expected: "30\n"},
	}
	res, err := Execute(context.Background(), code, "cpp", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictAccepted {
		t.Errorf("esperado accepted, got %s", res.Verdict)
	}
}

func TestJavaAccepted(t *testing.T) {
	code := `import java.util.Scanner;
public class Main {
    public static void main(String[] args) {
        Scanner sc = new Scanner(System.in);
        int a = sc.nextInt();
        int b = sc.nextInt();
        System.out.println(a+b);
    }
}`
	cases := []TestCase{
		{Input: "1 2\n", Expected: "3\n"},
		{Input: "10 20\n", Expected: "30\n"},
	}
	res, err := Execute(context.Background(), code, "java", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictAccepted {
		t.Errorf("esperado accepted, got %s", res.Verdict)
	}
}

func TestKotlinAccepted(t *testing.T) {
	code := `fun main() {
    val (a, b) = readLine()!!.split(" ").map { it.toInt() }
    println(a+b)
}`
	cases := []TestCase{
		{Input: "1 2\n", Expected: "3\n"},
		{Input: "10 20\n", Expected: "30\n"},
	}
	res, err := Execute(context.Background(), code, "kotlin", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictAccepted {
		t.Errorf("esperado accepted, got %s", res.Verdict)
	}
}

func TestJavaScriptAccepted(t *testing.T) {
	code := `const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });
rl.on('line', (line) => {
    const [a,b] = line.split(' ').map(Number);
    console.log(a+b);
    rl.close();
});`
	cases := []TestCase{
		{Input: "1 2\n", Expected: "3\n"},
		{Input: "10 20\n", Expected: "30\n"},
	}
	res, err := Execute(context.Background(), code, "javascript", cases, 2, 256)
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != VerdictAccepted {
		t.Errorf("esperado accepted, got %s", res.Verdict)
	}
}
