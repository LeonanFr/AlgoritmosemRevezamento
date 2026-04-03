import java.io.*;
import java.lang.reflect.Method;
import java.net.URL;
import java.net.URLClassLoader;
import java.nio.file.*;
import java.util.concurrent.*;
import javax.tools.JavaCompiler;
import javax.tools.ToolProvider;

public class Worker {
    private static final ExecutorService executor = Executors.newCachedThreadPool();

    public static void main(String[] args) throws Exception {
        BufferedReader in = new BufferedReader(new InputStreamReader(System.in));
        PrintWriter out = new PrintWriter(System.out, true);

        while (true) {
            String lang = in.readLine();
            StringBuilder codeBuilder = new StringBuilder();
            String line;
            while (!(line = in.readLine()).equals("---END_CODE---")) {
                codeBuilder.append(line).append("\n");
            }
            String code = codeBuilder.toString();

            int numCases = Integer.parseInt(in.readLine());
            String[] inputs = new String[numCases];
            String[] expected = new String[numCases];
            for (int i = 0; i < numCases; i++) {
                inputs[i] = in.readLine();
                expected[i] = in.readLine();
            }

            int timeLimit = Integer.parseInt(in.readLine());

            Path tmpDir = Files.createTempDirectory("worker_");
            try {
                boolean ok;
                if (lang.equals("java")) {
                    ok = processJava(tmpDir, code, inputs, expected, timeLimit, out);
                } else if (lang.equals("kotlin")) {
                    ok = processKotlin(tmpDir, code, inputs, expected, timeLimit, out);
                } else {
                    out.println("COMPILATION_ERROR");
                    out.println("Linguagem não suportada");
                    ok = false;
                }
                if (!ok) continue;
            } finally {
                deleteDirectory(tmpDir.toFile());
            }
        }
    }

    private static boolean processJava(Path tmpDir, String code, String[] inputs, String[] expected, int timeLimit, PrintWriter out) throws Exception {
        Path javaFile = tmpDir.resolve("Main.java");
        Files.write(javaFile, code.getBytes());

        JavaCompiler compiler = ToolProvider.getSystemJavaCompiler();
        int compileResult = compiler.run(null, null, null, "-d", tmpDir.toString(), javaFile.toString());
        if (compileResult != 0) {
            out.println("COMPILATION_ERROR");
            out.println("Erro de compilação Java");
            return false;
        }

        URLClassLoader classLoader = URLClassLoader.newInstance(new URL[]{tmpDir.toUri().toURL()});
        Class<?> mainClass = classLoader.loadClass("Main");
        Method mainMethod = mainClass.getMethod("main", String[].class);

        return runCases(mainMethod, inputs, expected, timeLimit, out);
    }

    private static boolean processKotlin(Path tmpDir, String code, String[] inputs, String[] expected, int timeLimit, PrintWriter out) throws Exception {
        Path ktFile = tmpDir.resolve("code.kt");
        Files.write(ktFile, code.getBytes());

        ProcessBuilder pb = new ProcessBuilder("kotlinc", ktFile.toString(), "-include-runtime", "-d", tmpDir.resolve("code.jar").toString());
        pb.directory(tmpDir.toFile());
        Process p = pb.start();
        int exitCode = p.waitFor();
        if (exitCode != 0) {
            out.println("COMPILATION_ERROR");
            out.println("Erro de compilação Kotlin");
            return false;
        }

        URLClassLoader classLoader = URLClassLoader.newInstance(new URL[]{tmpDir.resolve("code.jar").toUri().toURL()});
        Class<?> mainClass = classLoader.loadClass("CodeKt");
        Method mainMethod = mainClass.getMethod("main", String[].class);

        return runCases(mainMethod, inputs, expected, timeLimit, out);
    }

    private static boolean runCases(Method mainMethod, String[] inputs, String[] expected, int timeLimit, PrintWriter out) throws Exception {
        double totalTime = 0;
        for (int i = 0; i < inputs.length; i++) {
            final int caseIndex = i;
            final String input = inputs[i];
            final String expectedOutput = expected[i];

            FutureTask<String> task = new FutureTask<>(() -> {
                ByteArrayOutputStream baos = new ByteArrayOutputStream();
                System.setOut(new PrintStream(baos));
                System.setIn(new ByteArrayInputStream(input.getBytes()));

                long start = System.nanoTime();
                mainMethod.invoke(null, (Object) new String[0]);
                long elapsed = System.nanoTime() - start;
                double elapsedSec = elapsed / 1e9;

                String output = baos.toString().trim();
                return output + "\n" + elapsedSec;
            });

            executor.submit(task);
            try {
                String result = task.get(timeLimit, TimeUnit.SECONDS);
                String[] parts = result.split("\n", 2);
                String output = parts[0];
                double elapsedSec = Double.parseDouble(parts[1]);

                totalTime += elapsedSec;
                if (elapsedSec > timeLimit) {
                    out.println("TIME_LIMIT_EXCEEDED");
                    out.println(caseIndex);
                    return false;
                }

                if (!output.equals(expectedOutput.trim())) {
                    out.println("WRONG_ANSWER");
                    out.println(caseIndex);
                    return false;
                }
            } catch (TimeoutException e) {
                task.cancel(true);
                out.println("TIME_LIMIT_EXCEEDED");
                out.println(caseIndex);
                return false;
            } catch (Exception e) {
                out.println("RUNTIME_ERROR");
                out.println(caseIndex);
                return false;
            }
        }
        out.println("ACCEPTED");
        out.println(totalTime);
        return true;
    }

    private static void deleteDirectory(File dir) {
        File[] files = dir.listFiles();
        if (files != null) {
            for (File f : files) {
                if (f.isDirectory()) deleteDirectory(f);
                else f.delete();
            }
        }
        dir.delete();
    }
}