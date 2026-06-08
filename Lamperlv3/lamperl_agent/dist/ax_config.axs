/// Lamperl
let terminate_action  = menu.create_action("Terminate",  function(value) { value.forEach(v => ax.execute_command(v, "terminate")) });

menu.add_session_agent(terminate_action, ["Lamperl"])


let task_get_action = menu.create_action("Get task output", function(tasks_list) {
    tasks_list.forEach((task) => {
        if(task.state == "Running") {
            ax.execute_command(task.agent_id, "jobs get " + task.task_id);
        }
    });
});
menu.add_tasks(task_get_action, ["Lamperl"])

let task_stop_action = menu.create_action("Stop task", function(tasks_list) {
    tasks_list.forEach((task) => {
        if(task.state == "Running") {
            ax.execute_command(task.agent_id, "jobs kill " + task.task_id);
        }
    });
});
menu.add_tasks(task_stop_action, ["Lamperl"])



function RegisterCommands(listenerType)
{
/// Commands Here

    let cmd_pwd = ax.create_command("pwd", "Print working directory", "pwd", "Task: print working directory");
    
    let cmd_cd = ax.create_command("cd", "Change directory", "cd /etc", "Task: change directory");
    cmd_cd.addArgString("path", true, "Target directory path");
    

    let cmd_upload = ax.create_command("upload", "Upload files", "upload /local/path/file.txt /remote/path/file.txt", "Task: upload");
    cmd_upload.addArgFile("local_file", true);
    cmd_upload.addArgString("remote_path", true);

    let cmd_download = ax.create_command("download", "Download files", "download /path/file.txt", "Task: download");
    cmd_download.addArgString("file", true);


    let cmd_run = ax.create_command("run", "Execute command", "run whoami", "Task: execute command");
    cmd_run.addArgString("executable", true, "Command or executable to run");
    cmd_run.addArgString("args", false, "Command arguments");


    let cmd_terminate = ax.create_command("terminate", "Terminate beacon", "terminate", "Task: terminate beacon");

    let cmd_sleep = ax.create_command("sleep", "Sleep for a duration with optional jitter", "sleep 10 20", "Task: sleep");
    cmd_sleep.addArgInt("duration", true, "Duration to sleep in seconds");
    cmd_sleep.addArgInt("jitter", false, "Jitter percentage (0-100)");
    
    // Jobs subcommands
    let _cmd_job_get = ax.create_command("get", "Get job output", "jobs get 1", "Task: get job output");
    _cmd_job_get.addArgString("job_id", true, "Job ID to retrieve");
    _cmd_job_get.addArgInt("tail", false, "Return only last N lines (0 = all)");

    let _cmd_job_list = ax.create_command("list", "List of jobs", "jobs list", "Task: show jobs");

    let _cmd_job_kill = ax.create_command("kill", "Kill a specified job", "jobs kill 1a2b3c4d", "Task: kill job");
    _cmd_job_kill.addArgString("job_id", true, "Job ID to kill");

    let cmd_job = ax.create_command("jobs", "Long-running tasks manager");
    cmd_job.addSubCommands([_cmd_job_get, _cmd_job_list, _cmd_job_kill]);

    
    if(listenerType == "LamperlHTTP") {
        let commands_external = ax.create_commands_group("Lamperl", [cmd_pwd, cmd_cd, cmd_upload, cmd_download, cmd_run, cmd_terminate, cmd_sleep, cmd_job]);

        return { commands_linux: commands_external }
    }
    return ax.create_commands_group("none",[]);
}

function GenerateUI(listenerType)
{
    let container = form.create_container()

    let panel = form.create_panel()

    return {
        ui_panel: panel,
        ui_container: container
    }
}
