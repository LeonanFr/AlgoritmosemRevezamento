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
        System.out.println("READY");
        System.out.flush();

        BufferedReader in = new BufferedReader(new InputStreamReader(System.in));
        PrintWriter out = new PrintWriter(System.out, true);

        while (true) {
            String lang = in.readLine();
            if (lang == null) break;

            if ("PING".equals(lang)) {
                out.println("PONG");
                continue;
            }

            StringBuilder codeBuilder = new StringBuilder();
            String line;

            while ((line = in.readLine()) != null && !line.equals("---END_CODE---")) {
                codeBuilder.append(line).append("\n");
            }

            if (line == null) {
                break;
            }

            String code = codeBuilder.toString();

            String timeLimitLine = in.readLine();
            if (timeLimitLine == null) {
                break;
            }

            int timeLimit = Integer.parseInt(timeLimitLine);

            Path tmpDir = Files.createTempDirectory("worker_");

            try {
                Method mainMethod = null;

                if (lang.equals("java")) {
                    mainMethod = compileJava(tmpDir, code, out);
                } else if (lang.equals("kotlin")) {
                    mainMethod = compileKotlin(tmpDir, code, out);
                } else {
                    out.println("COMPILATION_ERROR");
                    out.println("Linguagem nao suportada");
                    out.println("---SEP---");
                    continue;
                }

                if (mainMethod == null) {
                    continue;
                }

                out.println("COMPILED_OK");

                while (true) {
                    String cmd = in.readLine();

                    if ("PING".equals(cmd)) {
                        out.println("PONG");
                        continue;
                    }

                    if (cmd == null || cmd.equals("STOP_CASES")) {
                        break;
                    }

                    if (cmd.equals("RUN_CASE")) {
                        String input = readUntilSep(in);
                        runSingleCase(mainMethod, input, timeLimit, out);
                    }
                }
            } finally {
                deleteDirectory(tmpDir.toFile());
            }
        }
    }

    private static String readUntilSep(BufferedReader in) throws IOException {
        StringBuilder sb = new StringBuilder();
        String line;
        while ((line = in.readLine()) != null && !line.equals("---SEP---")) {
            sb.append(line).append("\n");
        }
        return sb.toString();
    }

    private static Method compileJava(Path tmpDir, String code, PrintWriter out) throws Exception {
            Path javaFile = tmpDir.resolve("Main.java");
            Files.write(javaFile, code.getBytes());

            JavaCompiler compiler = ToolProvider.getSystemJavaCompiler();
            ByteArrayOutputStream errStream = new ByteArrayOutputStream();

            FutureTask<Integer> compileTask = new FutureTask<>(() ->
                    compiler.run(null, null, errStream, "-d", tmpDir.toString(), javaFile.toString())
            );
            executor.submit(compileTask);

            try {
                int compileResult = compileTask.get(30, TimeUnit.SECONDS);
                if (compileResult != 0) {
                    out.println("COMPILATION_ERROR");
                    out.println(errStream.toString());
                    out.println("---SEP---");
                    return null;
                }
            } catch (TimeoutException e) {
                compileTask.cancel(true);
                out.println("COMPILATION_ERROR");
                out.println("Tempo limite de compilacao excedido (30s).");
                out.println("---SEP---");
                return null;
            } catch (Exception e) {
                out.println("COMPILATION_ERROR");
                out.println("Erro interno do compilador.");
                out.println("---SEP---");
                return null;
            }

            URLClassLoader classLoader = URLClassLoader.newInstance(new URL[]{tmpDir.toUri().toURL()});
            Class<?> mainClass = classLoader.loadClass("Main");
            return mainClass.getMethod("main", String[].class);
        }

        private static Method compileKotlin(Path tmpDir, String code, PrintWriter out) throws Exception {
            Path ktFile = tmpDir.resolve("code.kt");
            Files.write(ktFile, code.getBytes());

            ProcessBuilder pb = new ProcessBuilder("kotlinc", ktFile.toString(), "-include-runtime", "-d", tmpDir.resolve("code.jar").toString());
            pb.directory(tmpDir.toFile());
            pb.redirectErrorStream(true);
            Process p = pb.start();

            FutureTask<String> readTask = new FutureTask<>(() -> {
                StringBuilder errSb = new StringBuilder();
                try (BufferedReader reader = new BufferedReader(new InputStreamReader(p.getInputStream()))) {
                    String line;
                    while ((line = reader.readLine()) != null) {
                        errSb.append(line).append("\n");
                    }
                }
                return errSb.toString();
            });
            executor.submit(readTask);

            boolean finished = p.waitFor(30, TimeUnit.SECONDS);
            if (!finished) {
                p.destroyForcibly();
                readTask.cancel(true);
                out.println("COMPILATION_ERROR");
                out.println("Tempo limite de compilacao excedido (30s).");
                out.println("---SEP---");
                return null;
            }

            int exitCode = p.exitValue();
            String errorOutput = readTask.get();
            if (exitCode != 0) {
                out.println("COMPILATION_ERROR");
                out.println(errorOutput);
                out.println("---SEP---");
                return null;
            }

            URLClassLoader classLoader = URLClassLoader.newInstance(new URL[]{tmpDir.resolve("code.jar").toUri().toURL()});
            Class<?> mainClass = classLoader.loadClass("CodeKt");
            return mainClass.getMethod("main", String[].class);
        }

    private static void runSingleCase(Method mainMethod, String input, int timeLimit, PrintWriter out) {
            FutureTask<String> task = new FutureTask<>(() -> {
                PrintStream originalOut = System.out;
                InputStream originalIn = System.in;
                ByteArrayOutputStream baos = new ByteArrayOutputStream();

                try {
                    System.setOut(new PrintStream(baos));
                    System.setIn(new ByteArrayInputStream(input.getBytes()));

                    long start = System.nanoTime();
                    mainMethod.invoke(null, (Object) new String[0]);
                    long elapsed = System.nanoTime() - start;
                    double elapsedSec = elapsed / 1e9;

                    return baos.toString() + "---TIME---" + elapsedSec;
                } finally {
                    System.setOut(originalOut);
                    System.setIn(originalIn);
                }
            });

            executor.submit(task);
            try {
                String result = task.get(timeLimit, TimeUnit.SECONDS);
                String[] parts = result.split("---TIME---");
                out.println("OK");
                out.println(parts[1]);
                out.print(parts[0]);
                out.flush();
                if (!parts[0].endsWith("\n") && !parts[0].isEmpty()) {
                    out.println();
                }
                out.println("---SEP---");
            } catch (TimeoutException e) {
                task.cancel(true);
                out.println("TIME_LIMIT_EXCEEDED");
                out.println("---SEP---");
            } catch (Exception e) {
                out.println("RUNTIME_ERROR");

                Throwable cause = e;
                while (cause instanceof ExecutionException || cause instanceof java.lang.reflect.InvocationTargetException) {
                    if (cause.getCause() != null) {
                        cause = cause.getCause();
                    } else {
                        break;
                    }
                }

                StringBuilder errSb = new StringBuilder();
                errSb.append(cause.toString()).append("\n");

                for (StackTraceElement ste : cause.getStackTrace()) {
                    String className = ste.getClassName();
                    if (className.equals("Worker")) {
                        break;
                    }
                    if (className.startsWith("jdk.internal.reflect") || className.startsWith("java.lang.reflect")) {
                        continue;
                    }
                    errSb.append("\tat ").append(ste.toString()).append("\n");
                }

                out.println(errSb.toString().trim());
                out.println("---SEP---");
            }
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