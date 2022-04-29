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
// using the signin command will prompt for username and password
$ sdpctl configure signin
? Username: <your username>
? Password: <your password>

// skip the prompting by setting the username and password as environment variables. This is only supported when using local provider for authentication.
$ SDPCTL_USERNAME=<username> SDPCTL_PASSWORD=<password> sdpctl configure signin

// setting only one of the environment variables will make the signin command prompt for the missing information. For example:
$ SDPCTL_USERNAME=<username> sdpctl configure signin
? Password: <password>
```

On successful authentication, a token is retrieved and stored in the sdpctl configuration and will be used for all the consecutive commands executed until the token expires. Once the token is expired, you'll need to re-authenticate to get a new token using the same signin command. For convenience, you can also store username and/or password for future use by using the `--remember-me` flag when logging in. Using this flag will delete any existing credentials that are already stored.

```bash
// Using credentials prompt
$ sdpctl configure signin --remember-me
? Username: user
? Password: ********
? What credentials should be saved? [Use arrows to move, type to filter]
> both
  only username
  only password

// Using environment variables
$ SDPCTL_USERNAME=<username> SDPCTL_PASSWORD=<password> sdpctl configure signin --remember-me
? What credentials should be saved? [Use arrows to move, type to filter]
> both
  only username
  only password
```

