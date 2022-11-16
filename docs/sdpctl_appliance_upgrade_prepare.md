## sdpctl appliance upgrade prepare

prepare appliances for upgrade

### Synopsis

Prepare an upgrade but do NOT install it.
This means the upgrade file will be downloaded/uploaded to all the appliances,
the signature verified as well as any other preconditions applicable at this point.

There are initial checks on the filename before attempting to upload it to the Appliances.
A valid filename ends with '.img.zip' and also needs to have a semver included somewhere
in the name, eg. 'upgrade.img.zip' will not not be valid, but 'upgrade5.5.3.img.zip' is
considered valid.

Note that the '--image' flag also accepts URL:s. The Appliances will then attempt to download
the upgrade image using the provided URL. It will fail if the Appliances cannot access the URL.

```
sdpctl appliance upgrade prepare [flags]
```

### Examples

```
  # prepare an upgrade from a local upgrade image
  > sdpctl appliance upgrade prepare --image=/path/to/upgrade-5.5.3.img.zip

  # prepare an upgrade from remote upgrade image
  > sdpctl appliance upgrade prepare --image=https://upgrade-host.com/upgrade-5.5.3.img.zip

  # use primary controller as an upgrade image host for the other appliances
  > sdpctl appliance upgrade prepare --image=https://upgrade-host.com/upgrade-5.5.3.img.zip --host-on-controller

  # prepare only certain appliances based on a filter
  > sdpctl appliance upgrade prepare --image=/path/to/image-5.5.3.img.zip --include function=controller
```

### Options

```
      --actual-hostname string   If the actual hostname is different from that which you are connecting to the appliance admin API, this flag can be used for setting the actual hostname.
      --dev-keyring              Use the development keyring to verify the upgrade image
      --force                    force prepare of upgrade on appliances even though the version uploaded is the same or lower then the version already running on the appliance
  -h, --help                     help for prepare
      --host-on-controller       Use primary controller as image host when uploading from remote source.
      --image string             Upgrade image file or URL
      --no-interactive           suppress interactive prompt with auto accept
      --throttle int             Upgrade is done in batches using a throttle value. You can control the throttle using this flag. (default 5)
```

### Options inherited from parent commands

```
      --api-version int          peer API version override
      --ci-mode                  log to stderr instead of file and disable progress-bars
      --debug                    Enable debug logging
  -e, --exclude stringToString   Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
                                 Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function (default [])
  -i, --include stringToString   Include appliances. Adheres to the same syntax and key-value pairs as '--exclude' (default [])
      --no-verify                don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string           profile configuration to use
  -t, --timeout duration         Timeout for the upgrade operation. The timeout applies to each appliance which is being operated on. (default 30m0s)
```

### SEE ALSO

* [sdpctl appliance upgrade](sdpctl_appliance_upgrade.md)	 - Perform appliance upgrade on the Appgate SDP Collective

