
## Example usage
### Single host (no forwarding)
This will start a simple logsuck instance which will index its own logs.

1. Create a configuration file named `logsuck.toml` in the working directory with the following contents:
    ```
    indexed_files = [ "./logsuck.log" ]
    ```
2. Run `logsuck > logsuck.log` in the working directory
3. Navigate to https://localhost:8080 to access the search interface
4. Try searching for `"Web GUI started"`

### Multiple hosts (with forwarding)
1. Create a directory called `recipient`. This directory will be the working directory for the logsuck instance which will receive logs and display the GUI.
2. Create a configuration file named `logsuck.toml` in the `recipient` directory with the following contents:
    ```
    [recipient]
    port = "9000"
    ```
3. Run the logsuck recipient instance by running `logsuck` in the `recipient` directory
4. Create a directory called `forwarded`. This folder will be the working directory for the logsuck instance which will forward logs to the recipient instance.
5. Create a configuration file named `logsuck.toml` in the `forwarded` directory with the following contents:
    ```
    indexed_files = [ "./logsuck.log" ]

    [forwarding]
    server = "127.0.0.1"
    port = "9000"
    ```
6. Run the logsuck forwarder by running `logsuck > logsuck.log` in the `forwarded` directory
7. Navigate to https://localhost:8080 to access the search interface
8. Try searching for `"Forwarding started"`

