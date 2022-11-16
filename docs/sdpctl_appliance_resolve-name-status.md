## sdpctl appliance resolve-name-status

Get the status of name resolution on a Gateway.

### Synopsis

Get the status of name resolution on a Gateway. It lists all the subscribed resource names from all the connected
Clients and shows the resolution results.

```
sdpctl appliance resolve-name-status [<appliance-id>] [flags]
```

### Examples

```
  # with a specific gateway appliance id
  > sdpctl appliance resolve-name-status 7f340572-0cd3-416b-7755-9f5c4e546391 --json
  {
      "resolutions": {
          "aws://lb-tag:kubernetes.io/service-name=opsnonprod/erp-dev": {
              "partial": false,
              "finals": [
                  "3.120.51.78",
                  "35.156.237.184"
              ],
              "partials": [
                  "dns://all.GW-ELB-2001535196.eu-central-1.elb.amazonaws.com",
                  "dns://all.purple-lb-1785267452.eu-central-1.elb.amazonaws.com"
              ],
              "errors": []
          }
      }
  }
```

### Options

```
  -h, --help   help for resolve-name-status
      --json   Display in JSON format
```

### Options inherited from parent commands

```
      --api-version int          peer API version override
      --ci-mode                  log to stderr instead of file and disable progress-bars
      --debug                    Enable debug logging
  -e, --exclude stringToString   Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
                                 Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function (default [])
  -i, --include stringToString   Include appliances. Adheres to the same syntax and key-value pairs as '--exclude' (default [])
      --no-interactive           suppress interactive prompt with auto accept
      --no-verify                don't verify TLS on for this particular command, overriding settings from config file
  -p, --profile string           profile configuration to use
```

### SEE ALSO

* [sdpctl appliance](sdpctl_appliance.md)	 - interact with Appgate SDP Appliances

