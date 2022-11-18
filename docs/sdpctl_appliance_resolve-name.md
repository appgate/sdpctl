## sdpctl appliance resolve-name

Test a resolver name on a Gateway

### Synopsis

Test a resolver name on a Gateway. Name resolvers are used by the Gateways on a Site resolve
the IPs in the specific network or set of protected resources.

```
sdpctl appliance resolve-name [<appliance-id>] [<query>] [flags]
```

### Examples

```
  # with a specific gateway appliance id
  > sdpctl appliance resolve-name d750ad44-7c6a-416d-773b-f805a2272418 dns://google.se

  # If you omit appliance id, you will be prompted with all online gateways, and you can select one to test on
  > sdpctl appliance resolve-name dns://google.se
  ? select appliance: gateway-9a9b8b70-faaa-4059-a061-761ce13783ba-site1 - Default Site - []
  142.251.36.3
  2a00:1450:400e:80f::2003
```

### Options

```
  -h, --help   help for resolve-name
      --json   Display in JSON format
```

### Options inherited from parent commands

```
      --api-version int          Peer API version override
      --ci-mode                  Log to stderr instead of file and disable progress-bars
      --debug                    Enable debug logging
  -e, --exclude stringToString   Filter appliances using a comma separated list of key-value pairs. Regex syntax is used for matching strings. Example: '--exclude name=controller,site=<site-id> etc.'.
                                 Available keywords to filter on are: name, id, tags|tag, version, hostname|host, active|activated, site|site-id, function (default [])
  -i, --include stringToString   Include appliances. Adheres to the same syntax and key-value pairs as '--exclude' (default [])
      --no-interactive           Suppress interactive prompt with auto accept
      --no-verify                Don't verify TLS on for the given command, overriding settings from config file
  -p, --profile string           Profile configuration to use
```

### SEE ALSO

* [sdpctl appliance](sdpctl_appliance.md)	 - Manage the appliances and perform tasks such as backups, ugprades, metrics etc

