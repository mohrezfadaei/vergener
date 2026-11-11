package semver

import (
	"fmt"
	"regexp"
	"strconv"
)

type Core struct{ Major, Minor, Patch int }

var pattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func Parse(s string) (Core, error) {
	m := pattern.FindStringSubmatch(s)
	if m == nil {
		return Core{}, fmt.Errorf("invalid stable version %q", s)
	}
	a, err := strconv.Atoi(m[1])
	if err != nil {
		return Core{}, fmt.Errorf("invalid major version %q: %w", m[1], err)
	}
	b, err := strconv.Atoi(m[2])
	if err != nil {
		return Core{}, fmt.Errorf("invalid minor version %q: %w", m[2], err)
	}
	c, err := strconv.Atoi(m[3])
	if err != nil {
		return Core{}, fmt.Errorf("invalid patch version %q: %w", m[3], err)
	}
	return Core{a, b, c}, nil
}
func (v Core) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }
func (v Core) Bump(kind string) (Core, error) {
	const maxInt = int(^uint(0) >> 1)
	switch kind {
	case "Major":
		if v.Major == maxInt {
			return Core{}, fmt.Errorf("major version overflow")
		}
		return Core{v.Major + 1, 0, 0}, nil
	case "Minor":
		if v.Minor == maxInt {
			return Core{}, fmt.Errorf("minor version overflow")
		}
		return Core{v.Major, v.Minor + 1, 0}, nil
	case "Patch":
		if v.Patch == maxInt {
			return Core{}, fmt.Errorf("patch version overflow")
		}
		return Core{v.Major, v.Minor, v.Patch + 1}, nil
	case "None":
		return v, nil
	default:
		return Core{}, fmt.Errorf("unresolved increment %q", kind)
	}
}
