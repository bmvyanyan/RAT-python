using System;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Text;
using System.Collections.Generic;

namespace CSharpShell
{
    internal static class Program
    {
        private static void Main(string[] args)
        {
            if (args.Length > 0)
            {
                ExecuteCommand(string.Join(' ', args));
                return;
            }

            Console.WriteLine("C# command shell. Type 'help' for available commands.");

            while (true)
            {
                Console.Write($"{Environment.CurrentDirectory}> ");
                var input = Console.ReadLine();
                if (input is null)
                    break;

                var trimmed = input.Trim();
                if (string.IsNullOrEmpty(trimmed))
                    continue;

                if (trimmed.Equals("exit", StringComparison.OrdinalIgnoreCase) || trimmed.Equals("quit", StringComparison.OrdinalIgnoreCase))
                    break;

                ExecuteCommand(trimmed);
            }
        }

        private static void ExecuteCommand(string input)
        {
            var args = SplitArguments(input);
            if (args.Count == 0)
                return;

            var command = args[0].ToLowerInvariant();
            switch (command)
            {
                case "help":
                    ShowHelp();
                    break;
                case "whoami":
                    ShowWhoAmI();
                    break;
                case "ps":
                    ListProcesses();
                    break;
                case "ls":
                    ListDirectory(args.Count > 1 ? args[1] : ".");
                    break;
                case "cd":
                    ChangeDirectory(args.Count > 1 ? args[1] : string.Empty);
                    break;
                case "pwd":
                    Console.WriteLine(Environment.CurrentDirectory);
                    break;
                case "download":
                    DownloadFile(args);
                    break;
                default:
                    Console.WriteLine($"Unknown command: {command}");
                    break;
            }
        }

        private static void ShowHelp()
        {
            Console.WriteLine("Available commands:");
            Console.WriteLine("  whoami                - Show the current user name");
            Console.WriteLine("  ps                    - List running processes");
            Console.WriteLine("  ls [path]             - List directory contents");
            Console.WriteLine("  cd <path>             - Change current directory");
            Console.WriteLine("  pwd                   - Print current directory");
            Console.WriteLine("  download <src> [dst]  - Copy a file from src to dst");
            Console.WriteLine("  help                  - Show this help text");
            Console.WriteLine("  exit, quit            - Exit the shell");
        }

        private static void ShowWhoAmI()
        {
            try
            {
                Console.WriteLine(Environment.UserName);
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error reading user name: {ex.Message}");
            }
        }

        private static void ListProcesses()
        {
            try
            {
                var processes = Process.GetProcesses()
                    .OrderBy(p => p.Id)
                    .Select(p => new { p.Id, p.ProcessName });

                Console.WriteLine("PID\tName");
                foreach (var process in processes)
                {
                    Console.WriteLine($"{process.Id}\t{process.ProcessName}");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error listing processes: {ex.Message}");
            }
        }

        private static void ListDirectory(string path)
        {
            try
            {
                var resolved = ResolvePath(path);
                if (File.Exists(resolved))
                {
                    Console.WriteLine(Path.GetFileName(resolved));
                    return;
                }

                if (!Directory.Exists(resolved))
                {
                    Console.WriteLine($"Directory not found: {resolved}");
                    return;
                }

                var directories = Directory.GetDirectories(resolved).OrderBy(d => d);
                var files = Directory.GetFiles(resolved).OrderBy(f => f);

                foreach (var directory in directories)
                    Console.WriteLine($"[D] {Path.GetFileName(directory)}");

                foreach (var file in files)
                    Console.WriteLine($"[F] {Path.GetFileName(file)}");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error listing directory: {ex.Message}");
            }
        }

        private static void ChangeDirectory(string path)
        {
            if (string.IsNullOrWhiteSpace(path))
            {
                Console.WriteLine("Usage: cd <path>");
                return;
            }

            try
            {
                var resolved = ResolvePath(path);
                if (!Directory.Exists(resolved))
                {
                    Console.WriteLine($"Directory not found: {resolved}");
                    return;
                }

                Environment.CurrentDirectory = resolved;
                Console.WriteLine(Environment.CurrentDirectory);
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error changing directory: {ex.Message}");
            }
        }

        private static void DownloadFile(IReadOnlyList<string> args)
        {
            if (args.Count < 2)
            {
                Console.WriteLine("Usage: download <sourcePath> [destinationPath]");
                return;
            }

            var sourcePath = ResolvePath(args[1]);
            var destinationPath = args.Count > 2 ? ResolvePath(args[2]) : Path.Combine(Environment.CurrentDirectory, Path.GetFileName(sourcePath));

            try
            {
                if (!File.Exists(sourcePath))
                {
                    Console.WriteLine($"Source file not found: {sourcePath}");
                    return;
                }

                var destinationDirectory = Path.GetDirectoryName(destinationPath);
                if (string.IsNullOrEmpty(destinationDirectory))
                    destinationDirectory = Environment.CurrentDirectory;

                if (!Directory.Exists(destinationDirectory))
                    Directory.CreateDirectory(destinationDirectory);

                File.Copy(sourcePath, destinationPath, overwrite: true);
                Console.WriteLine($"Downloaded '{sourcePath}' to '{destinationPath}'");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error downloading file: {ex.Message}");
            }
        }

        private static string ResolvePath(string path)
        {
            if (string.IsNullOrWhiteSpace(path))
                return Environment.CurrentDirectory;

            if (Path.IsPathRooted(path))
                return Path.GetFullPath(path);

            return Path.GetFullPath(Path.Combine(Environment.CurrentDirectory, path));
        }

        private static List<string> SplitArguments(string commandLine)
        {
            var args = new List<string>();
            if (string.IsNullOrWhiteSpace(commandLine))
                return args;

            var current = new StringBuilder();
            var inQuotes = false;
            for (var i = 0; i < commandLine.Length; i++)
            {
                var c = commandLine[i];
                if (c == '"')
                {
                    inQuotes = !inQuotes;
                    continue;
                }

                if (!inQuotes && char.IsWhiteSpace(c))
                {
                    if (current.Length > 0)
                    {
                        args.Add(current.ToString());
                        current.Clear();
                    }
                    continue;
                }

                current.Append(c);
            }

            if (current.Length > 0)
                args.Add(current.ToString());

            return args;
        }
    }
}
