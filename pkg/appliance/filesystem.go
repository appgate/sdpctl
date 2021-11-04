package appliance

import "os"

func IsOnAppliance() bool {
	if _, err := os.Stat("/mnt/state/config"); os.IsNotExist(err) {
		return false
	}
	return true
}
