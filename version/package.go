package version // import "github.com/cloudfoundry-incubator/golang-bump-progress/version"

import (
	"fmt"
	"strings"
)

func MajorMinor(version string) string {
	parts := strings.Split(version, ".")
	return fmt.Sprintf("%s.%s", parts[0], parts[1])
}
