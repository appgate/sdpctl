<p align="center">
	<img src="./appgate.svg" width="200">
</p>

# Quick Start

Recommended starting point for admins is the [Quick Start Guide](https://appgate.github.io/sdpctl).

# Introduction

An Appgate SDP Collective can be managed by a number of different means. Prior to v6.0 scripts were used for a number of these tasks. From v6.0 a new command line tool "sdpctl" has been introduced for managing various aspects of your Appgate SDP Collective. The most critical of these being backups (of the Controller) and upgrades (of the Collective). sdpctl is the recommended tool for managing these aspects of SDP once you are running v6.0.

Over time we will add more features to sdpctl so please be sure to always use the latest version.

# Installation

## Signature verification
Before installation make sure the verify the signature of the downloaded binaries.
Release binary checksums are signed using a GPG key, the [public key](https://bin.appgate-sdp.com/appgate-inc.pub) with key id `5635CFCADCF8A718`.

To import and trust the key:
```bash
wget https://bin.appgate-sdp.com/appgate-inc.pub
gpg --import appgate-inc.pub
gpg --edit-key 5635CFCADCF8A718
gpg> trust
gpg> 5
gpg> quit
```

The `checksums.txt.asc` contains the signature for `checksums.txt` as well as its content.
On Linux you can verify the checksums signature as well as the checksums of the binaries using the following command:
```bash
gpg --output - --verify checksums.txt.asc | sha256sum --check --ignore-missing
```

## macOS
Download the latest [macOS build](https://github.com/appgate/sdpctl/releases/latest) from the releases page. Then install using the command line:
```bash
# Unpack the downloaded package in the current directory
$ gunzip -c <path-to-downloaded-tar> | tar xopf -

# Install the binary
$ sudo mv <binary-path> /usr/local/bin/sdpctl
$ sudo chmod 0755 /usr/local/bin/sdpctl
```

## Windows
Download the latest [Windows build](https://github.com/appgate/sdpctl/releases/latest) from the releases page. Install using the command line:
```powershell
# Create a folder for the binary
PS> mkdir <folder-path>

# Unzip the downloaded archive
PS> Expand-Archive <path-to-archive> -DestinationPath <folder-path>

# Edit the PATH
PS> [Environment]::SetEnvironmentVariable("PATH", $Env:Path + ";<folder-path>", [EnvironmentVariableTarget]::Machine)
```
Then restart Powershell to make the changes take effect.

## Linux
**Debian based distributions:**
Download the latest [debian package](https://github.com/appgate/sdpctl/releases/latest) from the releases. Then install it:
```bash
$ sudo dpkg -i <path-to-downloaded-debian-package>
```

**Red Hat/Fedora:**
Download the latest [rpm package](https://github.com/appgate/sdpctl/releases/latest) from the releases page. Then install using this command:
```bash
$ sudo rpm -i <path-to-downloaded-rpm-package>
```

**Binary (Cross-platform)**
Download the appropriate version for your platform from sdpctl [releases](https://github.com/appgate/sdpctl/releases/latest). Once downloaded, the binary can be run from anywhere. You don’t need to install it into a specific location. This works well for shared hosts and other systems where you don’t have a privileged account.

Ideally, you should install it somewhere in your PATH for easy use. /usr/local/bin is the most probable location.

# Shell completion

The `sdpctl` tool supports shell completions for `bash`, `zsh`, `fish` and `PowerShell`. See the completion help command for more information on shell completions:
```
$ sdpctl completion --help
```
