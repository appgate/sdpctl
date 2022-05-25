# Configuring `sdpctl`

For using sdpctl, you first need to configure and authenticate to the Appgate SDP collective. You'll need the url for the controller you'd like to connect to, as well as a username and password (currently only local provider is supported). Configure sdpctl to connect to your Appgate SDP collective by running `sdpctl configure` and responding to the prompts:
```bash
$ sdpctl configure
? Enter the url for the controller API (example https://appgate.controller.com/admin) https://sdp.controller.com/admin
```
Optionally, if the controller uses an unsigned certificate, you can trust the certificate by specifying a PEM file for the command to use for certificate verification. You can that by using the `--pem` flag on the configure command:

```bash
$ sdpctl configure --pem=<path/to/pem>
```

After the host and TLS verification options are set, you'll need to authenticate to the controller:

```bash
$ # using the signin command will prompt for username and password
$ sdpctl configure signin
? Username: <your username>
? Password: <your password>

$ # skip the prompting by setting the username and password as environment variables. This is only supported when using local provider for authentication.
$ SDPCTL_USERNAME=<username> SDPCTL_PASSWORD=<password> sdpctl configure signin

$ # setting only one of the environment variables will make the signin command prompt for the missing information. For example:
$ SDPCTL_USERNAME=<username> sdpctl configure signin
? Password: <password>
```

```

## Working with multiple appgate sdp collectives

sdpctl support working with multiple appgate sdp collectives. Using the environment variable `SDPCTL_CONFIG_DIR` you can toggle which collective you want
to work against. If `SDPCTL_CONFIG_DIR` is not set by the user, the default directory will be `$XDG_CONFIG_HOME/sdpctl` or `$HOME/.config/sdpctl` on UNIX and `%APPDATA%\Local\sdpctl` on Windows.

Imagine you have the following file structure, where each directory represent a appgatesdp collective.


```bash
$ cd ~/.config/sdpctl
$ tree .
.
├── acme
│  ├── ca-cert.pem
│  ├── config.json
├── daily-bugle
│  ├── ca-cert.pem
│  ├── config.json
├── oscorp
│  └── config.json
└── stark-industries
   └── config.json

```


```bash
$ # export SDPCTL_CONFIG_DIR to the acme directory to use this config.
$ export SDPCTL_CONFIG_DIR=$HOME/.config/sdpctl/acme
```

```bash
$ # appliance list for acme
$ sdpctl appliance list
Name                                                   ID                                    Hostname                 Site          Activated
----                                                   --                                    --------                 ----          ---------
controller-2eedefae-ad19-4367-b1e3-f4b688997bdf-site1  ec36a6f2-cd61-42a4-8791-d0bfd3a460bb  envy-10-97-180-2.devops  Default Site  true
gateway-2eedefae-ad19-4367-b1e3-f4b688997bdf-site1     7f340572-0cd3-416b-7755-9f5c4e546391  envy-10-97-180-3.devops  Default Site  true
```


```bash
$ # if we want to swap to antoher collective, export SDPCTL_CONFIG_DIR to another config directory.
$ export SDPCTL_CONFIG_DIR=$HOME/.config/sdpctl/daily-bugle
```

```bash
$ # appliance list for daily bugle
$ sdpctl appliance list
Name                                                   ID                                    Hostname                                    Site          Activated
----                                                   --                                    --------                                    ----          ---------
controller-abb9ce60-8711-4361-8b97-50f3ff9c2199-site1  4c519e1c-d5f2-4241-97d5-1ae8219175d1  ec2-3-86-111-140.compute-1.amazonaws.com    Default Site  true
gateway-abb9ce60-8711-4361-8b97-50f3ff9c2199-site1     857f194d-75e8-4d3b-68b1-5897dce4fb18  ec2-54-175-105-232.compute-1.amazonaws.com  Default Site  true

```
