<img src="./appgate_sdp_logo.svg" width="400">

---
# sdpctl documentation
sdpctl is a command line tool for managing your Appgate SDP collective.

---
# Installing

## Linux
**Debian based distributions:**
Download the latest [debian package](https://github.com/appgate/sdpctl/releases/latest) from the releases. Then install it:
```bash
$ sudo dpkg -i <path-to-downloaded-debian-package>
```

**Red Hat/Fedora:**
Download the latest [rpm package](https://github.com/appgate/sdpctl/releases/latest) from the releases. Then install using this command:
```bash
$ sudo rpm -i <path-to-downloaded-rpm-package>
```

**Binary (Cross-platform)**
Download the appropriate version for your platform from sdpctl [releases](https://github.com/appgate/sdpctl/releases/latest). Once downloaded, the binary can be run from anywhere. You don’t need to install it into a global location. This works well for shared hosts and other systems where you don’t have a privileged account.

Ideally, you should install it somewhere in your PATH for easy use. /usr/local/bin is the most probable location.

## MacOS
Download the latest [darwin binary](https://github.com/appgate/sdpctl/releases/latest) from the releases. Then install using the command line:
```bash
# Unpack the downloaded package in the current directory
$ gunzip -c <path-to-downloaded-tar> | tar xopf -

# Install the binary
$ sudo mv <binary-path> /usr/local/bin/sdpctl
$ sudo chown root:root /usr/local/bin/sdpctl
```

## Windows
Download the latest [windows build](https://github.com/appgate/sdpctl/releases/latest) from the releases page. Install using the command line:
```powershell
# Create a folder for the binary
$ mkdir <folder-path>

# Unzip the downloaded archive
$ Expand-Archive <path-to-archive> -DestinationPath <folder-path>

# Edit the PATH for your account
$ setx PATH "%PATH;<folder-path>"
```
Then restart Powershell to make the changes take effect.

# Shell completion
The `sdpctl` tool supports shell completions for `bash`, `zsh`, `fish` and `PowerShell`. See the completion help command for more information on shell completions:
```
$ sdpctl completion --help
```

---
# Usage
## Initial setup
To start using `sdpctl`, you'll need to authenticate with your Appgate SDP collective. The authentication process is a two step process where you first configure `sdpctl` and then authenticate by signing in to the collective configured in the first step.

See the [configuration documentation](./docs/configure.md) for a more detailed description on how to use the configure command.

Example:
```bash
# Initial configuration
$ sdpctl configure
? Enter the url for the controller API (example https://appgate.controller.com/admin)

# Sign in
$ sdpctl configure signin
```

You can also manage multiple Appgate SDP collectives using `sdpctl`. See the [authentication documentation](./docs/auth.md) for more information.

---
## The `appliance` command
The `appliance` command is the base command in `sdpctl` for managing appliance resource specific tasks, such as backing up appliances or upgrading them. The appliance command requires at least one action command following it. Executing the appliance command without an action command will print the help text for the command.

See the [appliance command documentation](./docs/appliance.md) for a more detailed description

### Examples
```bash
# Listing appliances
$ sdpctl appliance list

# Backing up appliances
$ sdpctl appliance backup

# Upgrading appliances
$ sdpctl appliance upgrade prepare --image=<appliance-image>
$ sdpctl appliance upgrade complete
```

---
## The `token` command


---
## Global flags
| Flag | Shorthand | Description |
|---|---|---|
| `--api-version` | none | peer API version override |
| `--debug` | none | Enable debug output and logging |
| `--help` | `-h` | Help for sdpctl and any subcommand of it |
| `--version` | `-v` | Current version of sdpctl |
| `--no-verify` | none | don't verify TLS on for this particular command, overriding settings from config file. USE WITH CAUTION! |

