package memory

// NOTE(Friso): I genuinely dispise how this is implemented right now, it should be changed

import (
	"fmt"
	"strings"
)

type PathLevel int

const (
	PathLevelInvalid PathLevel = iota
	PathLevelWorld
	PathLevelSector
	PathLevelArena
	PathLevelObject
)

type PathState int

const (
	PathStateNormal PathState = iota

	pathStateSpecialStart

	PathStateWaiting
	PathStatePersona
	PathStateSpawningLocation

	pathstateSpecialEnd

	PathStateRandom
)

const (
	WaitingArgFormat = "X: %d, Y: %d"
)

func (s PathState) ToString() string {
	switch s {
	case PathStateNormal:
		return ""
	case PathStatePersona:
		return "<persona>"
	case PathStateRandom:
		return "<random>"
	case PathStateWaiting:
		return "<waiting>"
	case PathStateSpawningLocation:
		return "<spawn_loc>"
	default:
		panic(fmt.Errorf("unexpected memory.PathState: %#v", s))
	}
}

type PathOption func(*Path)

func PathWithWorld(world string) PathOption   { return func(p *Path) { p.world = world } }
func PathWithSector(sector string) PathOption { return func(p *Path) { p.sector = sector } }
func PathWithArena(arena string) PathOption   { return func(p *Path) { p.arena = arena } }
func PathWithObject(object string) PathOption { return func(p *Path) { p.object = object } }

// Path in the file system sense to an object in the game world,
// name should probagly be different but it's the best I can come
// up with right now
type Path struct {
	// NOTE(Friso): This implementation _should_ be changed but this is roughly how they implemented it in the original code with some QOL additions
	world  string
	sector string
	arena  string
	object string
}

func ParsePath(loc string) Path {
	parts := strings.Split(loc, ":")
	if len(parts) > 4 {
		panic("paths should consist of 1-4 parts separated by ':'")
	}

	l := Path{world: parts[0]}

	if len(parts) > 1 {
		l.sector = parts[1]
	}
	if len(parts) > 2 {
		l.arena = parts[2]
	}
	if len(parts) > 3 {
		l.object = parts[3]
	}

	return l
}

func NewPath(opts ...PathOption) Path {
	p := Path{}

	for _, opt := range opts {
		opt(&p)
	}

	return p
}

func SpecialPath(state PathState, arg string) Path {
	switch state {
	case PathStateNormal:
		return ParsePath(arg)
	case PathStateRandom:
		return ParsePath(arg).
			Copy(PathWithObject("<random>"))
	default:
		return ParsePath(fmt.Sprintf("%s %s", state.ToString(), arg))
	}
}

func (p Path) Copy(opts ...PathOption) Path {
	newPath := p

	for _, opt := range opts {
		opt(&newPath)
	}

	return newPath
}

func (p Path) ToString() string {
	str := p.world
	if p.sector == "" {
		return str
	}

	str += ":" + p.sector
	if p.arena == "" {
		return str
	}

	str += ":" + p.arena
	if p.object == "" {
		return str
	}

	str += ":" + p.object
	return str
}

func (p Path) HasState(state PathState) bool {
	return p.Contains(state.ToString())
}

func (p Path) Contains(substr string) bool {
	// NOTE(Friso): When updating the path representation make sure to fix this function
	return strings.Contains(p.world, substr) ||
		strings.Contains(p.sector, substr) ||
		strings.Contains(p.arena, substr) ||
		strings.Contains(p.object, substr)
}

func (p Path) Base() string {
	if p.object != "" {
		return p.object
	}
	if p.arena != "" {
		return p.arena
	}
	if p.sector != "" {
		return p.sector
	}
	return p.world
}

func (p Path) Get(level PathLevel) string {
	switch level {
	case PathLevelWorld:
		return p.world
	case PathLevelSector:
		return p.sector
	case PathLevelArena:
		return p.arena
	case PathLevelObject:
		return p.object
	default:
		panic(fmt.Errorf("trying to get path with invalid level: %d", level))
	}
}

func (p Path) AtLevel(level PathLevel) (newPath Path) {
	if level == PathLevelInvalid {
		return p
	}

	if level >= PathLevelWorld {
		newPath.world = p.world
	}
	if level >= PathLevelSector {
		newPath.sector = p.sector
	}
	if level >= PathLevelArena {
		newPath.arena = p.arena
	}
	if level >= PathLevelObject {
		newPath.object = p.object
	}

	return
}

func (p Path) Level() PathLevel {
	if p.sector == "" {
		return PathLevelWorld
	} else if p.arena == "" {
		return PathLevelSector
	} else if p.object == "" {
		return PathLevelArena
	} else {
		return PathLevelObject
	}
}

func (p Path) Matches(mask Path) bool {
	if mask.world != "" && p.world != mask.world {
		return false
	}
	if mask.sector != "" && p.sector != mask.sector {
		return false
	}
	if mask.arena != "" && p.arena != mask.arena {
		return false
	}
	if mask.object != "" && p.object != mask.object {
		return false
	}
	return true
}

func (p Path) IsEmpty() bool {
	return p.world == "" &&
		p.sector == "" &&
		p.arena == "" &&
		p.object == ""
}

func (p Path) GetArg() string {
	for i := pathStateSpecialStart + 1; i < pathstateSpecialEnd; i += 1 {
		prefix := i.ToString()
		if strings.HasPrefix(p.world, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(p.world, prefix))
		}
	}

	return ""
}

func (p Path) IsSpecial(s PathState) bool {
	switch s {
	case PathStatePersona:
		return strings.HasPrefix(p.world, "<persona>")
	case PathStateRandom:
		return p.Level() == PathLevelObject && p.object == "<random>"
	case PathStateWaiting:
		return strings.HasPrefix(p.world, "<waiting>")
	default:
		return false
	}
}

func (p Path) IsObject() bool {
	return p.object != ""
}
