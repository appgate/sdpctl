# Appgatectl

<img src="./appgate_sdp_logo.svg" width="400">

---
Appgatectl is a command line tool for managing your Appgate SDP collective.

## Installation

### Linux
**Debian based distributions:**

Download the latest [debian package](https://github.com/appgate/appgatectl/releases/latest) from the releases.

**Red Hat/Fedora:**

**Binary (Cross-platform)**

Download the appropriate version for your platform from appgatectl [releases](https://github.com/appgate/appgatectl/releases/latest). Once downloaded, the binary can be run from anywhere. You don’t need to install it into a global location. This works well for shared hosts and other systems where you don’t have a privileged account.

Ideally, you should install it somewhere in your PATH for easy use. /usr/local/bin is the most probable location.
**Others:**



### Windows


## Usage

### Initial setup
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

# alternatively you can skip the prompt by setting the username and password as environment varibles
$ APPGATECTL_USERNAME=<username> APPGATECTL_PASSWORD=<password> appgatectl configure login

# setting only one of the environment variables will make the login command prompt for the missing information. For example:
$ APPGATECTL_USERNAME=<username> appgatectl configure login
? Password: <password>
```

On successful authentication, a token is retrieved and stored in the appgatectl configuration and will be used for all the consecutive commands executed until the token expires. Once the token is expired, you'll need to re-authenticate to get a new token using the same login command. For convenience, you can also store username and/or password for future use by using the `--remember-me` flag when logging in.

```bash
$ APPGATECTL_USERNAME=<username> APPGATECTL_PASSWORD=<password> appgatectl configure login --remember-me
```
