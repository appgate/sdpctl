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
The token command let's you list and revoke device tokens according to the options specified. To revoke a single token, use these commands:
```bash
# List the current tokens
$ sdpctl token list
Distinguished Name                                     Device ID                             Last Token Issued At         Provider Name  Username
------------------                                     ---------                             --------------------         -------------  --------
CN=8401189b492f4d76b6671a9ba03b4ce1,CN=admin,OU=local  8401189b-492f-4d76-b667-1a9ba03b4ce1  2022-02-21T07:22:12.375464Z  local          admin

# Revoke the token
$ sdpctl token revoke CN=8401189b492f4d76b6671a9ba03b4ce1,CN=admin,OU=local
```

More details on the token command can be found in [the token command documentation](./docs/token.md)

---
## Other available commands

### `sdpctl open`
The open command will attempt to open the Appgate SDP Collective administration interface in the systems default browser.

### `sdpctl help [command]`
The help command will print the help page for any command that follows.

---
## Global flags
| Flag | Shorthand | Description |
|---|---|---|
| `--api-version` | none | peer API version override |
| `--debug` | none | Enable debug output and logging |
| `--no-verify` | none | Don't verify TLS on for this particular command, overriding settings from config file. USE WITH CAUTION! |
