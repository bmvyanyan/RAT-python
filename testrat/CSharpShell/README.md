# C# Command Shell

This is a simple C# console application that accepts commands from the terminal and supports:

- `whoami` — prints the current user name
- `ps` — lists running processes
- `ls [path]` — lists files and directories
- `cd <path>` — changes current directory
- `download <source> [destination]` — copies a file to a destination path
- `help` — shows available commands
- `exit` or `quit` — ends the shell

## Build

```bash
dotnet build
```

## Run

```bash
dotnet run --project CSharpShell/CSharpShell.csproj
```

Or use an argument string directly:

```bash
dotnet run --project CSharpShell/CSharpShell.csproj -- "ls ."
```
