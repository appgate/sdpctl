# Authentication



## Working with multiple appgate sdp collectives

sdpctl support working with multiple appgate sdp collectives. Using the environment variable `SDPCTL_CONFIG_DIR` you can toggle which collective you want
to work against. If `SDPCTL_CONFIG_DIR` is not set by the user, the default directory will be `$XDG_CONFIG_HOME/sdpctl` or `$HOME/.config/sdpctl` on UNIX and `%APPDATA%/sdpctl` on Windows.

Imagine you have the following file structure, where each directory represent a appgatesdp collective.


```bash
~/.config/sdpctl
> tree .
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
> # export SDPCTL_CONFIG_DIR to the acme directory to use this config.
> export SDPCTL_CONFIG_DIR=$HOME/.config/sdpctl/acme
```

```bash
> # appliance list for acme
> sdpctl appliance list
Name                                                   ID                                    Hostname                 Site          Activated
----                                                   --                                    --------                 ----          ---------
controller-2eedefae-ad19-4367-b1e3-f4b688997bdf-site1  ec36a6f2-cd61-42a4-8791-d0bfd3a460bb  envy-10-97-180-2.devops  Default Site  true
gateway-2eedefae-ad19-4367-b1e3-f4b688997bdf-site1     7f340572-0cd3-416b-7755-9f5c4e546391  envy-10-97-180-3.devops  Default Site  true
```


```bash
# if we want to swap to antoher collective, export SDPCTL_CONFIG_DIR to another config directory.
> export SDPCTL_CONFIG_DIR=$HOME/.config/sdpctl/daily-bugle
```

```bash
> # appliance list for daily bugle
> sdpctl appliance list
Name                                                   ID                                    Hostname                                    Site          Activated
----                                                   --                                    --------                                    ----          ---------
controller-abb9ce60-8711-4361-8b97-50f3ff9c2199-site1  4c519e1c-d5f2-4241-97d5-1ae8219175d1  ec2-3-86-111-140.compute-1.amazonaws.com    Default Site  true
gateway-abb9ce60-8711-4361-8b97-50f3ff9c2199-site1     857f194d-75e8-4d3b-68b1-5897dce4fb18  ec2-54-175-105-232.compute-1.amazonaws.com  Default Site  true

```
