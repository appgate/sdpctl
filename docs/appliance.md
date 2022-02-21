# the `appliance` command
The `appliance` command is the base command in `sdpctl` for managing appliance resource specific tasks, such as backing up appliances or upgrading them. The appliance command requires at least one action command following it. Executing the appliance command without an action command will print the help text for the command.

### Available actions:
- [list](#listing-appliances)
- [backup](#backing-up-appliances)
- [upgrade](#upgrading-appliances)
- [metric](#monitoring-appliances)
- [stats](#monitoring-appliances)
- [resolve-name](#name-resolving)
- [resolve-name-status](#name-resolving)

### Flags:
| Flag | Shorthand | Description | Syntax | Default |
|---|---|---|---|---|
| `--filter` | `f` | Filter appliances that should be included in the command | Filter appliances using a comma seperated list of key-value pairs. Example: `--filter name=controller,site=<site-id>` etc. Available keywords to filter on are: **name**, **id**, **tags\|tag**, **version**, **hostname\|host**, **active\|activated**, **site\|site-id**, **function\|roles\|role** | null |
| `--exclude` | `-e` | The opposite of the filter flag, but uses the same syntax | Se syntax description on the `--filter` flag | null |
| `--no-interactive` | none | Using this flag will attempt to skip all user interaction otherwise required by accepting the default values | `sdpctl appliance --no-interactive [action]` | null |

---
## Listing appliances
You can get a list of all appliances by using the `list` command.
```bash
$ sdpctl appliance list
Name                    ID                                        Hostname                      Site          Activated
----                    --                                        --------                      ----          ---------
controller2-site1       5c587cce-3032-42a3-8d04-df08c356fb39      controller2.yoursite.com      Default Site  true
controller-site1        972d1887-9e30-4233-b6ea-a7042ee1dc5e      controller.yoursite.com       Default Site  true
gateway-site1           c8c80704-e65a-44d0-430b-ab997d288159      gateway.yoursite.com          Default Site  true
```

---
## Backing up appliances
For backing up appliances, you can use the `sdpctl appliance backup` command. Using the backup command will send a backup request to the selected appliances and result in a backup file being downloaded for each backed up appliance.

Using the backup command requires the backup API on the appliance to be enabled. If the backup API is disabled, you can also enable it by running this command and set a password for the backup API:
```bash
$ sdpctl appliance backup api
```

Using the backup command without any arguments or flags will prompt for what appliances to backup.
```bash
$ sdpctl appliance backup
? select appliances to backup:  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
> [ ]  controller2-site1
  [ ]  controller-site1
  [ ]  gateway-site1
```

You can also specify what to backup by using one or more appliance names as arguments to the backup command:
```bash
# Will backup controller-site1
$ sdpctl appliance backup controller-site1

# Will backup controller-site1 and gateway-site1
$ sdpctl appliance backup controller-site1 gateway-site1
```

There are also flags to help select what appliances to backup. The `--primary` flag will find the primary controller in the collective and perform a backup of that. Similarly, the `--current` flag performs a backup of the appliance which sdpctl is currently connected to. The `--all` flag will perform a backup of all appliances in the collective.

You can also select appliances to backup using the global `--filter` flag and the backup will be performed only on the appliances that match the filter query. On the opposite, if you'd want to exclude some specific appliances from the backup, you can use the `--exclude` flag. The exclude flag uses the same syntax as the global filter flag. When both the `--filter` and `--exclude` flags are used combined, the exclusion will apply after the filtering. In other words, the exclusion will apply to the list of appliances that matches the filtering rules.
```bash
# given that our list of appliances is the same as provided in the list command example, this command will only backup the controller-site1 appliance
$ sdpctl appliance backup --filter function=controller --exclude name=controller2
```

The backups will be downloaded to a provided destination on your filesystem. The default destination is in the `Download` folder of the user home directory, eg. `$HOME/Downloads/appgate/backups`. You can define a custom destination for downloading the backups by providing the `--destination` flag when running the backup command. The user executing the script will need permission to write to that folder.
```bash
$ sdpctl appliance backup --destination /your/custom/backup/destination
```

---
## Upgrading appliances
You can use `sdpctl` for upgrading your Appgate SDP appliances using the `upgrade` action command. Upgrading is a two step process where you first need to upload an image of the newer version which you want to upgrade to. You can find all supported Appgate SDP images available on [Appgate SDP support page](https://www.appgate.com/support/software-defined-perimeter-support).

> Note: You can use the `upgrade` command along with the `--filter` and/or `--exclude` flags. This will upgrade only the appliances matching the filter or exclude query.

You can view the current status of an upgrade by running `upgrade status`. If no upgrade is in progress, the upgrade status should be 'idle':
```bash
$ sdpctl appliance upgrade status
ID                                          Name                    Status        Upgrade Status        Details
04cee88e-64bb-4389-adc0-ad01e752a001        controller-site1        online        idle
47e9e708-0a9b-484d-b356-0b8f38cb13ec        controller2-site1       online        idle
15786382-501a-4185-6713-d6a57e8f1448        gateway-site1           online        idle
```

Once you have an image to upgrade your appliances with, you upload it using the `upgrade prepare` command. The `prepare` command has a mandatory `--image` flag where you will specify the path to the image you want to upload.
```bash
$ sdpctl appliance upgrade prepare --image /path/to/image-5.5.3.img.zip
```
> Note: The image path could either be a local file path to a downloaded image or a URL.<br />
> If the path is a URL, make sure the URL is accessible so that the appliances can download it.

Once the `upgrade prepare` command is completed, the upgrade status of the appliances should now be 'ready' and the 'Details' column should have the filename on the uploaded file:
```bash
$ sdpctl appliance upgrade status
ID                                          Name                    Status        Upgrade Status        Details
04cee88e-64bb-4389-adc0-ad01e752a001        controller-site1        online        ready                 image-5.5.3.img.zip
47e9e708-0a9b-484d-b356-0b8f38cb13ec        controller2-site1       online        ready                 image-5.5.3.img.zip
15786382-501a-4185-6713-d6a57e8f1448        gateway-site1           online        ready                 image-5.5.3.img.zip
```

At this point, you can choose to abort the upgrade using the `upgrade cancel` command. Running the `cancel` command will remove the uploaded upgrade image and return the appliances to the 'idle' state.

If you wish to continue upgrading, the upgrade is completed using the `upgrade complete` command.
```bash
$ sdpctl appliance upgrade complete
```
At this point, you will be prompted if you want to do a backup before proceeding to complete the upgrade. If you want more backup options than provided in the prompt, it's recommended to use the standalone `appliance backup` command, since more options are available there.

The `upgrade complete` command will run until all appliances that are part of the upgrade reaches the desired state of 'idle'.

## Monitoring appliances
There are two commands in `sdpctl` to help monitoring appliances: `metric` and `stats`.

The `stats` command will print out system resource statistics as well as some other useful information on each specific appliance.
```bash
$ sdpctl appliance stats
Name                          Status         Function                      CPU         Memory        Network out/in             Disk        Version
controller-site1              healthy        log server, controller        0.1%        50.8%         43.2 bps / 48.0 bps        1.4%        5.5.3-27108-release
gateway-site1                 healthy        gateway                       0.3%        8.1%          43.3 bps / 48.1 bps        0.7%        5.5.2-27039-release
```

The `stats` command also accepts a `--json` flag, which will print out a more detailed information view in json format.

TODO: Add description for metrics command

## Name resolving

