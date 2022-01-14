<img src="./appgate_sdp_logo.svg" width="400">

---
# appgatectl documentation
Appgatectl is a command line tool for managing your Appgate SDP collective.

---
# Installation

## Linux
**Debian based distributions:**
Download the latest [debian package](https://github.com/appgate/appgatectl/releases/latest) from the releases.

**Red Hat/Fedora:**


**Binary (Cross-platform)**
Download the appropriate version for your platform from appgatectl [releases](https://github.com/appgate/appgatectl/releases/latest). Once downloaded, the binary can be run from anywhere. You don’t need to install it into a global location. This works well for shared hosts and other systems where you don’t have a privileged account.

Ideally, you should install it somewhere in your PATH for easy use. /usr/local/bin is the most probable location.

## MacOS


## Windows


---
# Usage
## Initial setup
For using appgatectl, you first need to configure and authenticate to the Appgate SDP collective. You'll need the url for the controller you'd like to connect to, as well as a username and password (currently only local provider is supported). Configure appgatectl to connect to your Appgate SDP collective by running `appgatectl configure` and responding to the prompts:
```shell
$ appgatectl configure
? Enter the url for the controller API (example https://appgate.controller.com/admin) https://sdp.controller.com/admin
? Whether server should be accessed without verifying the TLS certificate true
```
In case you chose to access the controller with TLS verification, you also need to provide a path to a valid PEM file.

After the host and TLS verification options are set, you'll need to authenticate to the controller:

```bash
# using the login command will prompt for username and password
$ appgatectl configure login
? Username: <your username>
? Password: <your password>

# skip the prompting by setting the username and password as environment varibles. This is only supported when using local provider for authentication.
$ APPGATECTL_USERNAME=<username> APPGATECTL_PASSWORD=<password> appgatectl configure login

# setting only one of the environment variables will make the login command prompt for the missing information. For example:
$ APPGATECTL_USERNAME=<username> appgatectl configure login
? Password: <password>
```

On successful authentication, a token is retrieved and stored in the appgatectl configuration and will be used for all the consecutive commands executed until the token expires. Once the token is expired, you'll need to re-authenticate to get a new token using the same login command. For convenience, you can also store username and/or password for future use by using the `--remember-me` flag when logging in.

```bash
$ APPGATECTL_USERNAME=<username> APPGATECTL_PASSWORD=<password> appgatectl configure login --remember-me
```
---
## The `appliance` command
The `appliance` command is the base command in `appgatectl` for managing appliance resource specific tasks, such as backing up appliances or upgrading them. The appliance command requires at least one action command following it. Executing the appliance command without an action command will print the help text for the command.

Available actions for the appliance command:
- [list](#listing-appliances)
- [backup](#backing-up-appliances)
- [upgrade](#upgrading-appliances)

---
### Listing appliances
You can get a list of all appliances by using the `list` command.
```bash
$ appgatectl appliance list
Name                Hostname                  Site          Activated
----                --------                  ----          ---------
controller2-site1   controller2.yoursite.com  Default Site  true
controller-site1    controller.yoursite.com   Default Site  true
gateway-site1       gateway.yoursite.com      Default Site  true
```

---
### Backing up appliances
For backing up appliances, you can use the `appgatectl appliance backup` command. Using the backup command will send a backup request to the selected appliances and result in a backup file being downloaded for each backed up appliance.

Using the backup command requires the backup API on the appliance to be enabled. If the backup API is disabled, you can also enable it by running this command and set a password for the backup API:
```bash
$ appgatectl appliance backup api
```

Using the backup command without any arguments or flags will prompt for what appliances to backup.
```bash
$ appgatectl appliance backup
? select appliances to backup:  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
> [ ]  controller2-site1
  [ ]  controller-site1
  [ ]  gateway-site1
```

You can also specify what to backup by using one or more appliance names as arguments to the backup command:
```bash
# Will backup controller-site1
$ appgatectl appliance backup controller-site1

# Will backup controller-site1 and gateway-site1
$ appgatectl appliance backup controller-site1 gateway-site1
```

There are also flags to help select what appliances to backup. The `--primary` flag will find the primary controller in the collective and perform a backup of that. Similarly, the `--current` flag performs a backup of the appliance which appgatectl is currently connected to. The `--all` flag will perform a backup of all appliances in the collective.

You can also select appliances to backup using the global `--filter` flag and the backup will be performed only on the appliances that match the filter query. On the opposite, if you'd want to exclude some specific appliances from the backup, you can use the `--exclude` flag. The exclude flag uses the same syntax as the global filter flag. When both the `--filter` and `--exclude` flags are used combined, the exclusion will apply after the filtering. In other words, the exclusion will apply to the list of appliances that matches the filtering rules.
```bash
# given that our list of appliances is the same as provided in the list command example, this command will only backup the controller-site1 appliance
$ appgatectl appliance backup --filter function=controller --exclude name=controller2
```

The backups will be downloaded to a provided destination on your filesystem. The default destination is in the `Download` folder of the user home directory, eg. `$HOME/Downloads/appgate/backups`. You can define a custom destination for downloading the backups by providing the `--destination` flag when running the backup command. The user executing the script will need permission to write to that folder.
```bash
$ appgatectl appliance backup --destination /your/custom/backup/destination
```

---
### Upgrading appliances
You can use `appgatectl` for upgrading your Appgate SDP appliances using the `upgrade` action command. Upgrading is a two step process where you first need to upload an image of the newer version which you want to upgrade to. You can find all supported Appgate SDP images available on [Appgate SDP support page](https://www.appgate.com/support/software-defined-perimeter-support).

Once you have an image to upgrade your appliances with, you upload it using the `upgrade prepare` command. The `prepare` command has a mandatory `--image` flag where you will specify the path to the image you want to upload.
```bash
$ appgatectl appliance upgrade prepare --image /path/to/image.img.zip
```



---
## The `token` command


---
## Global flags
